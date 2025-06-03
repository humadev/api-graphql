package main

import (
	"context"
	"fmt"
	"log"
	"net/http" // Ditambahkan untuk HTTP server
	"sync"

	"github.com/google/uuid"
	// Ganti dengan path package yang dihasilkan oleh protoc
	pb "example.com/graphql-app/grpc" // Sesuaikan path ini
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Impor untuk gRPC-Web
	"github.com/improbable-eng/grpc-web/go/grpcweb"
)

// --- Penyimpanan Data In-Memory (Sama seperti contoh sebelumnya) ---
var (
	mahasiswaStore  = make(map[string]*pb.Mahasiswa)
	mataKuliahStore = make(map[string]*pb.MataKuliah)
	mahasiswaLock   sync.RWMutex
	mataKuliahLock  sync.RWMutex
)

// --- Implementasi Server (akademikServer tetap sama) ---
type akademikServer struct {
	pb.UnimplementedAkademikServiceServer
}

// ... (Semua implementasi metode RPC: CreateMahasiswa, GetMahasiswa, dll. tetap SAMA seperti contoh gRPC sebelumnya) ...
// Pastikan fungsi protoDeepCopyMahasiswa() juga ada jika Anda menggunakannya.

// Fungsi utilitas untuk deep copy pesan Mahasiswa (penting agar tidak memodifikasi state di map secara tidak sengaja)
func protoDeepCopyMahasiswa(src *pb.Mahasiswa) *pb.Mahasiswa {
	if src == nil {
		return nil
	}
	dst := &pb.Mahasiswa{
		Id:            src.Id,
		Nim:           src.Nim,
		Nama:          src.Nama,
		Jurusan:       src.Jurusan,
		MataKuliahIds: make([]string, len(src.MataKuliahIds)),
		// DetailMataKuliah akan diisi terpisah jika diperlukan
	}
	copy(dst.MataKuliahIds, src.MataKuliahIds)
	if src.DetailMataKuliah != nil {
		dst.DetailMataKuliah = make([]*pb.MataKuliah, len(src.DetailMataKuliah))
		for i, mk := range src.DetailMataKuliah {
			// Lakukan deep copy untuk MataKuliah juga jika perlu,
			// tapi untuk contoh ini, kita anggap MataKuliah cukup sederhana.
			dst.DetailMataKuliah[i] = &pb.MataKuliah{
				Id:     mk.Id,
				KodeMk: mk.KodeMk,
				NamaMk: mk.NamaMk,
				Sks:    mk.Sks,
			}
		}
	}
	return dst
}

// Implementasi Metode RPC Mahasiswa (Contoh singkat, implementasi penuh ada di contoh gRPC sebelumnya)
func (s *akademikServer) CreateMahasiswa(ctx context.Context, req *pb.CreateMahasiswaRequest) (*pb.Mahasiswa, error) {
	log.Printf("Menerima permintaan CreateMahasiswa: %v", req)
	mahasiswaLock.Lock()
	defer mahasiswaLock.Unlock()

	id := uuid.New().String()
	mhs := &pb.Mahasiswa{
		Id:            id,
		Nim:           req.GetNim(),
		Nama:          req.GetNama(),
		Jurusan:       req.GetJurusan(),
		MataKuliahIds: []string{},
	}
	mahasiswaStore[id] = mhs
	log.Printf("Mahasiswa dibuat: %v", mhs)
	return mhs, nil
}

func (s *akademikServer) GetMahasiswa(ctx context.Context, req *pb.GetMahasiswaRequest) (*pb.Mahasiswa, error) {
	log.Printf("Menerima permintaan GetMahasiswa: %v", req)
	mahasiswaLock.RLock()
	mhs, ok := mahasiswaStore[req.GetId()]
	mahasiswaLock.RUnlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "Mahasiswa dengan ID %s tidak ditemukan", req.GetId())
	}
	responseMhs := protoDeepCopyMahasiswa(mhs)
	if req.GetSertakanDetailMataKuliah() {
		mataKuliahLock.RLock()
		defer mataKuliahLock.RUnlock()
		responseMhs.DetailMataKuliah = make([]*pb.MataKuliah, 0, len(mhs.GetMataKuliahIds()))
		for _, mkId := range mhs.GetMataKuliahIds() {
			if mk, mkOk := mataKuliahStore[mkId]; mkOk {
				responseMhs.DetailMataKuliah = append(responseMhs.DetailMataKuliah, mk)
			}
		}
	} else {
		responseMhs.DetailMataKuliah = nil
	}
	log.Printf("Mengembalikan mahasiswa: %v", responseMhs)
	return responseMhs, nil
}
func (s *akademikServer) ListMahasiswa(ctx context.Context, req *pb.ListMahasiswaRequest) (*pb.ListMahasiswaResponse, error) {
	mahasiswaLock.RLock()
	defer mahasiswaLock.RUnlock()
	var mhsList []*pb.Mahasiswa
	for _, mhs := range mahasiswaStore {
		copiedMhs := protoDeepCopyMahasiswa(mhs)
		if req.GetSertakanDetailMataKuliah() {
			mataKuliahLock.RLock()
			copiedMhs.DetailMataKuliah = make([]*pb.MataKuliah, 0, len(mhs.GetMataKuliahIds()))
			for _, mkId := range mhs.GetMataKuliahIds() {
				if mk, mkOk := mataKuliahStore[mkId]; mkOk {
					copiedMhs.DetailMataKuliah = append(copiedMhs.DetailMataKuliah, mk)
				}
			}
			mataKuliahLock.RUnlock()
		} else {
			copiedMhs.DetailMataKuliah = nil
		}
		mhsList = append(mhsList, copiedMhs)
	}
	return &pb.ListMahasiswaResponse{DaftarMahasiswa: mhsList}, nil
}
func (s *akademikServer) CreateMataKuliah(ctx context.Context, req *pb.CreateMataKuliahRequest) (*pb.MataKuliah, error) {
	mataKuliahLock.Lock()
	defer mataKuliahLock.Unlock()
	id := uuid.New().String()
	mk := &pb.MataKuliah{Id: id, KodeMk: req.GetKodeMk(), NamaMk: req.GetNamaMk(), Sks: req.GetSks()}
	mataKuliahStore[id] = mk
	return mk, nil
}
func (s *akademikServer) ListMataKuliah(ctx context.Context, req *pb.ListMataKuliahRequest) (*pb.ListMataKuliahResponse, error) {
	mataKuliahLock.RLock()
	defer mataKuliahLock.RUnlock()
	var mkList []*pb.MataKuliah
	for _, mk := range mataKuliahStore {
		mkList = append(mkList, mk)
	}
	return &pb.ListMataKuliahResponse{DaftarMataKuliah: mkList}, nil
}
func (s *akademikServer) DaftarkanMataKuliahUntukMahasiswa(ctx context.Context, req *pb.DaftarkanMataKuliahRequest) (*pb.Mahasiswa, error) {
	mhsID := req.GetMahasiswaId()
	mkID := req.GetMataKuliahId()
	mahasiswaLock.Lock()
	defer mahasiswaLock.Unlock()
	mhs, ok := mahasiswaStore[mhsID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Mahasiswa %s tidak ditemukan", mhsID)
	}
	mataKuliahLock.RLock()
	_, mkOk := mataKuliahStore[mkID]
	mataKuliahLock.RUnlock()
	if !mkOk {
		return nil, status.Errorf(codes.NotFound, "Mata Kuliah %s tidak ditemukan", mkID)
	}
	for _, existingMkID := range mhs.GetMataKuliahIds() {
		if existingMkID == mkID {
			return protoDeepCopyMahasiswa(mhs), status.Errorf(codes.AlreadyExists, "Sudah terdaftar")
		}
	}
	mhs.MataKuliahIds = append(mhs.MataKuliahIds, mkID)
	mahasiswaStore[mhsID] = mhs
	responseMhs := protoDeepCopyMahasiswa(mhs)
	mataKuliahLock.RLock()
	responseMhs.DetailMataKuliah = make([]*pb.MataKuliah, 0, len(mhs.GetMataKuliahIds()))
	for _, id := range mhs.GetMataKuliahIds() {
		if mkDetail, exists := mataKuliahStore[id]; exists {
			responseMhs.DetailMataKuliah = append(responseMhs.DetailMataKuliah, mkDetail)
		}
	}
	mataKuliahLock.RUnlock()
	return responseMhs, nil
}

// Implementasi RPC lainnya (UpdateMahasiswa, DeleteMahasiswa, GetMataKuliah) disingkat untuk brevity
// Anda harus memiliki implementasi penuh dari contoh gRPC sebelumnya.
func (s *akademikServer) UpdateMahasiswa(ctx context.Context, req *pb.UpdateMahasiswaRequest) (*pb.Mahasiswa, error) { /* ... */
	return nil, status.Errorf(codes.Unimplemented, "method UpdateMahasiswa not implemented")
}
func (s *akademikServer) DeleteMahasiswa(ctx context.Context, req *pb.DeleteMahasiswaRequest) (*pb.EmptyResponse, error) { /* ... */
	return nil, status.Errorf(codes.Unimplemented, "method DeleteMahasiswa not implemented")
}
func (s *akademikServer) GetMataKuliah(ctx context.Context, req *pb.GetMataKuliahRequest) (*pb.MataKuliah, error) { /* ... */
	return nil, status.Errorf(codes.Unimplemented, "method GetMataKuliah not implemented")
}

func main() {
	// Buat server gRPC seperti biasa
	grpcServer := grpc.NewServer()
	pb.RegisterAkademikServiceServer(grpcServer, &akademikServer{})

	// Bungkus server gRPC dengan gRPC-Web handler
	// Opsi ini mengizinkan semua origin, cocok untuk development.
	// Untuk produksi, konfigurasikan origin yang diizinkan.
	wrappedGrpc := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true // Izinkan semua origin untuk development
		}))

	// Buat HTTP server untuk melayani gRPC-Web
	// gRPC-Web biasanya dilayani melalui HTTP/1.1 atau HTTP/2
	// Port ini berbeda dari port gRPC murni jika Anda ingin menjalankannya secara terpisah.
	// Untuk kesederhanaan, kita akan melayani gRPC-Web di port yang berbeda (misalnya :8080)
	// sementara gRPC murni bisa tetap di :50051 jika diperlukan.
	// Atau, Anda bisa menggunakan satu port jika dikonfigurasi dengan benar.

	httpServer := &http.Server{
		// Handler ini akan menangani permintaan gRPC-Web.
		// Permintaan non-gRPC-Web lainnya akan menghasilkan 404 atau bisa diarahkan ke handler lain.
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			// Tambahkan header CORS secara manual jika diperlukan,
			// meskipun WithOriginFunc seharusnya sudah menangani ini untuk gRPC-Web.
			// Untuk menyajikan file statis (HTML, JS klien), Anda akan memerlukan handler lain.
			log.Printf("Menerima permintaan HTTP: %s %s", req.Method, req.URL.Path)
			if wrappedGrpc.IsGrpcWebRequest(req) || wrappedGrpc.IsAcceptableGrpcCorsRequest(req) || wrappedGrpc.IsGrpcWebSocketRequest(req) {
				wrappedGrpc.ServeHTTP(resp, req)
				return
			}
			// Tangani permintaan non-gRPC-Web lainnya di sini (misalnya, menyajikan HTML klien)
			// Untuk contoh ini, kita akan tambahkan handler file statis di bawah.
			http.NotFound(resp, req)
		}),
		Addr: fmt.Sprintf(":%d", 8080), // Port untuk gRPC-Web & HTTP
	}

	log.Printf("Server gRPC-Web (dan HTTP) berjalan di http://localhost:8080")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Gagal menjalankan server HTTP: %v", err)
	}

	// Kode untuk menjalankan server gRPC murni (jika masih ingin menjalankannya secara terpisah di port lain):
	/*
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Gagal membuka port gRPC murni: %v", err)
		}
		log.Printf("Server gRPC murni berjalan di port %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil { // Catatan: grpcServer, bukan s dari NewServer()
			log.Fatalf("Gagal menjalankan server gRPC murni: %v", err)
		}
	*/
}

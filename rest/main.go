package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- Model Data ---

// Mahasiswa struct
type Mahasiswa struct {
	ID           string   `json:"id"`
	NIM          string   `json:"nim"`
	Nama         string   `json:"nama"`
	Jurusan      string   `json:"jurusan"`
	MataKuliahID []string `json:"mataKuliahId,omitempty"` // ID mata kuliah yang diambil
}

// MataKuliah struct
type MataKuliah struct {
	ID     string `json:"id"`
	KodeMK string `json:"kodeMk"`
	NamaMK string `json:"namaMk"`
	SKS    int    `json:"sks"`
}

// --- Penyimpanan Data In-Memory (untuk contoh) ---
var (
	mahasiswaStore  = make(map[string]Mahasiswa)
	mataKuliahStore = make(map[string]MataKuliah)
	mahasiswaLock   sync.RWMutex
	mataKuliahLock  sync.RWMutex
)

func initData() {
	mk1 := MataKuliah{ID: uuid.New().String(), KodeMK: "IF101", NamaMK: "Dasar Pemrograman", SKS: 3}
	mk2 := MataKuliah{ID: uuid.New().String(), KodeMK: "IF102", NamaMK: "Struktur Data", SKS: 4}
	mk3 := MataKuliah{ID: uuid.New().String(), KodeMK: "UM101", NamaMK: "Bahasa Indonesia", SKS: 2}

	mataKuliahStore[mk1.ID] = mk1
	mataKuliahStore[mk2.ID] = mk2
	mataKuliahStore[mk3.ID] = mk3

	mhs1 := Mahasiswa{
		ID:           uuid.New().String(),
		NIM:          "2023001",
		Nama:         "Adi Nugraha",
		Jurusan:      "Teknik Informatika",
		MataKuliahID: []string{mk1.ID, mk2.ID},
	}
	mhs2 := Mahasiswa{
		ID:           uuid.New().String(),
		NIM:          "2023002",
		Nama:         "Siti Aminah",
		Jurusan:      "Sistem Informasi",
		MataKuliahID: []string{mk1.ID, mk3.ID},
	}
	mahasiswaStore[mhs1.ID] = mhs1
	mahasiswaStore[mhs2.ID] = mhs2
}

// --- Handler untuk Mahasiswa ---

// CreateMahasiswa membuat mahasiswa baru
func CreateMahasiswa(c *gin.Context) {
	var mhs Mahasiswa
	if err := c.ShouldBindJSON(&mhs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mhs.ID = uuid.New().String() // Generate ID unik

	mahasiswaLock.Lock()
	mahasiswaStore[mhs.ID] = mhs
	mahasiswaLock.Unlock()

	c.JSON(http.StatusCreated, mhs)
}

// GetMahasiswa mengambil semua mahasiswa
func GetMahasiswa(c *gin.Context) {
	mahasiswaLock.RLock()
	defer mahasiswaLock.RUnlock()

	mhsList := make([]Mahasiswa, 0, len(mahasiswaStore))
	for _, mhs := range mahasiswaStore {
		mhsList = append(mhsList, mhs)
	}
	c.JSON(http.StatusOK, mhsList)
}

// GetMahasiswaByID mengambil mahasiswa berdasarkan ID
func GetMahasiswaByID(c *gin.Context) {
	id := c.Param("id")
	mahasiswaLock.RLock()
	mhs, ok := mahasiswaStore[id]
	mahasiswaLock.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mahasiswa tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, mhs)
}

// UpdateMahasiswa memperbarui data mahasiswa
func UpdateMahasiswa(c *gin.Context) {
	id := c.Param("id")
	var updatedMhs Mahasiswa
	if err := c.ShouldBindJSON(&updatedMhs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mahasiswaLock.Lock()
	defer mahasiswaLock.Unlock()
	_, ok := mahasiswaStore[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mahasiswa tidak ditemukan"})
		return
	}
	updatedMhs.ID = id // Pastikan ID tidak berubah
	mahasiswaStore[id] = updatedMhs
	c.JSON(http.StatusOK, updatedMhs)
}

// DeleteMahasiswa menghapus mahasiswa
func DeleteMahasiswa(c *gin.Context) {
	id := c.Param("id")
	mahasiswaLock.Lock()
	defer mahasiswaLock.Unlock()
	if _, ok := mahasiswaStore[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mahasiswa tidak ditemukan"})
		return
	}
	delete(mahasiswaStore, id)
	c.JSON(http.StatusOK, gin.H{"message": "Mahasiswa berhasil dihapus"})
}

// TambahMataKuliahUntukMahasiswa mendaftarkan mata kuliah untuk mahasiswa
func TambahMataKuliahUntukMahasiswa(c *gin.Context) {
	mhsID := c.Param("id")
	var reqBody struct {
		MataKuliahID string `json:"mataKuliahId"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	mahasiswaLock.Lock()
	defer mahasiswaLock.Unlock()
	mataKuliahLock.RLock()
	defer mataKuliahLock.RUnlock()

	mhs, mhsExists := mahasiswaStore[mhsID]
	if !mhsExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mahasiswa tidak ditemukan"})
		return
	}

	_, mkExists := mataKuliahStore[reqBody.MataKuliahID]
	if !mkExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mata Kuliah tidak ditemukan"})
		return
	}

	// Cek apakah mata kuliah sudah diambil
	for _, idMk := range mhs.MataKuliahID {
		if idMk == reqBody.MataKuliahID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mahasiswa sudah mengambil mata kuliah ini"})
			return
		}
	}

	mhs.MataKuliahID = append(mhs.MataKuliahID, reqBody.MataKuliahID)
	mahasiswaStore[mhsID] = mhs
	c.JSON(http.StatusOK, mhs)
}

// --- Handler untuk Mata Kuliah ---

// CreateMataKuliah membuat mata kuliah baru
func CreateMataKuliah(c *gin.Context) {
	var mk MataKuliah
	if err := c.ShouldBindJSON(&mk); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mk.ID = uuid.New().String()

	mataKuliahLock.Lock()
	mataKuliahStore[mk.ID] = mk
	mataKuliahLock.Unlock()

	c.JSON(http.StatusCreated, mk)
}

// GetMataKuliah mengambil semua mata kuliah
func GetMataKuliah(c *gin.Context) {
	mataKuliahLock.RLock()
	defer mataKuliahLock.RUnlock()

	mkList := make([]MataKuliah, 0, len(mataKuliahStore))
	for _, mk := range mataKuliahStore {
		mkList = append(mkList, mk)
	}
	c.JSON(http.StatusOK, mkList)
}

// GetMataKuliahByID mengambil mata kuliah berdasarkan ID
func GetMataKuliahByID(c *gin.Context) {
	id := c.Param("id")
	mataKuliahLock.RLock()
	mk, ok := mataKuliahStore[id]
	mataKuliahLock.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mata Kuliah tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, mk)
}

func main() {
	initData()
	r := gin.Default()

	// Endpoint Mahasiswa
	mahasiswaRoutes := r.Group("/mahasiswa")
	{
		mahasiswaRoutes.POST("", CreateMahasiswa)
		mahasiswaRoutes.GET("", GetMahasiswa)
		mahasiswaRoutes.GET("/:id", GetMahasiswaByID)
		mahasiswaRoutes.PUT("/:id", UpdateMahasiswa)
		mahasiswaRoutes.DELETE("/:id", DeleteMahasiswa)
		// Endpoint untuk menghubungkan mahasiswa dengan mata kuliah (edge)
		mahasiswaRoutes.POST("/:id/matakuliah", TambahMataKuliahUntukMahasiswa)
	}

	// Endpoint Mata Kuliah
	mataKuliahRoutes := r.Group("/matakuliah")
	{
		mataKuliahRoutes.POST("", CreateMataKuliah)
		mataKuliahRoutes.GET("", GetMataKuliah)
		mataKuliahRoutes.GET("/:id", GetMataKuliahByID)
		// Tambahkan PUT dan DELETE jika diperlukan
	}

	// Representasi "Canvas" (Hubungan):
	// Untuk melihat mata kuliah yang diambil seorang mahasiswa:
	// GET /mahasiswa/:id akan mengembalikan Mahasiswa beserta MataKuliahID
	// Klien kemudian bisa melakukan GET /matakuliah/:id_mk untuk detail setiap mata kuliah.
	// Atau, kita bisa membuat endpoint khusus:
	r.GET("/mahasiswa/:id/detailmatakuliah", func(c *gin.Context) {
		mhsID := c.Param("id")

		mahasiswaLock.RLock()
		mhs, okMhs := mahasiswaStore[mhsID]
		mahasiswaLock.RUnlock()

		if !okMhs {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mahasiswa tidak ditemukan"})
			return
		}

		detailMataKuliah := []MataKuliah{}
		mataKuliahLock.RLock()
		for _, mkID := range mhs.MataKuliahID {
			if mk, okMk := mataKuliahStore[mkID]; okMk {
				detailMataKuliah = append(detailMataKuliah, mk)
			}
		}
		mataKuliahLock.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"mahasiswa":          mhs,
			"matakuliah_diambil": detailMataKuliah,
		})
	})

	r.Run(":8080") // Jalankan server di port 8080
}

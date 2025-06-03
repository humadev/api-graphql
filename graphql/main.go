package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
)

// --- Model Data (Sama seperti contoh REST) ---
type Mahasiswa struct {
	ID           string   `json:"id"`
	NIM          string   `json:"nim"`
	Nama         string   `json:"nama"`
	Jurusan      string   `json:"jurusan"`
	MataKuliahID []string `json:"-"` // Kita akan resolve ini menjadi objek MataKuliah
}

type MataKuliah struct {
	ID     string `json:"id"`
	KodeMK string `json:"kodeMk"`
	NamaMK string `json:"namaMk"`
	SKS    int    `json:"sks"`
}

// --- Penyimpanan Data In-Memory (Sama seperti contoh REST) ---
var (
	mahasiswaStore  = make(map[string]Mahasiswa)
	mataKuliahStore = make(map[string]MataKuliah)
	mahasiswaLock   sync.RWMutex
	mataKuliahLock  sync.RWMutex
)

// --- Inisialisasi Data Awal (Contoh) ---
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

// --- GraphQL Type Definitions ---
var mataKuliahType *graphql.Object
var mahasiswaType *graphql.Object

func defineTypes() {
	mataKuliahType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "MataKuliah",
			Fields: graphql.Fields{
				"id":     &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"kodeMk": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"namaMk": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"sks":    &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			},
		},
	)

	mahasiswaType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Mahasiswa",
			Fields: graphql.Fields{
				"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"nim":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"nama":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"jurusan": &graphql.Field{Type: graphql.String},
				"mataKuliah": &graphql.Field{ // Ini adalah "edge" atau hubungan ke MataKuliah
					Type: graphql.NewList(mataKuliahType),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						if mhs, ok := p.Source.(Mahasiswa); ok {
							mataKuliahLock.RLock()
							defer mataKuliahLock.RUnlock()
							var result []MataKuliah
							for _, mkID := range mhs.MataKuliahID {
								if mk, exists := mataKuliahStore[mkID]; exists {
									result = append(result, mk)
								}
							}
							return result, nil
						}
						return nil, nil
					},
				},
			},
		},
	)
}

// --- GraphQL Schema ---
var schema graphql.Schema

func buildSchema() {
	defineTypes() // Pastikan tipe sudah didefinisikan

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"mahasiswa": &graphql.Field{
				Type: mahasiswaType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					idQuery, _ := params.Args["id"].(string)
					mahasiswaLock.RLock()
					defer mahasiswaLock.RUnlock()
					if mhs, ok := mahasiswaStore[idQuery]; ok {
						return mhs, nil
					}
					return nil, fmt.Errorf("Mahasiswa dengan ID %s tidak ditemukan", idQuery)
				},
			},
			"semuaMahasiswa": &graphql.Field{
				Type: graphql.NewList(mahasiswaType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					mahasiswaLock.RLock()
					defer mahasiswaLock.RUnlock()
					list := make([]Mahasiswa, 0, len(mahasiswaStore))
					for _, mhs := range mahasiswaStore {
						list = append(list, mhs)
					}
					return list, nil
				},
			},
			"mataKuliah": &graphql.Field{
				Type: mataKuliahType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					idQuery, _ := params.Args["id"].(string)
					mataKuliahLock.RLock()
					defer mataKuliahLock.RUnlock()
					if mk, ok := mataKuliahStore[idQuery]; ok {
						return mk, nil
					}
					return nil, fmt.Errorf("Mata Kuliah dengan ID %s tidak ditemukan", idQuery)
				},
			},
			"semuaMataKuliah": &graphql.Field{
				Type: graphql.NewList(mataKuliahType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					mataKuliahLock.RLock()
					defer mataKuliahLock.RUnlock()
					list := make([]MataKuliah, 0, len(mataKuliahStore))
					for _, mk := range mataKuliahStore {
						list = append(list, mk)
					}
					return list, nil
				},
			},
		},
	})

	// Input type untuk menambah mahasiswa
	mahasiswaInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "MahasiswaInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"nim":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"nama":    &graphql.InputObjectFieldConfig{Type: graphql.String},
			"jurusan": &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	})

	// Input type untuk menambah mata kuliah
	mataKuliahInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "MataKuliahInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"kodeMk": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"namaMk": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"sks":    &graphql.InputObjectFieldConfig{Type: graphql.Int},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootMutation",
		Fields: graphql.Fields{
			"tambahMahasiswa": &graphql.Field{
				Type: mahasiswaType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(mahasiswaInputType),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					input, _ := params.Args["input"].(map[string]interface{})
					newMhs := Mahasiswa{
						ID:           uuid.New().String(),
						NIM:          input["nim"].(string),
						Nama:         input["nama"].(string),
						Jurusan:      input["jurusan"].(string),
						MataKuliahID: []string{}, // Default kosong
					}
					mahasiswaLock.Lock()
					mahasiswaStore[newMhs.ID] = newMhs
					mahasiswaLock.Unlock()
					return newMhs, nil
				},
			},
			"tambahMataKuliah": &graphql.Field{
				Type: mataKuliahType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(mataKuliahInputType),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					input, _ := params.Args["input"].(map[string]interface{})
					newMk := MataKuliah{
						ID:     uuid.New().String(),
						KodeMK: input["kodeMk"].(string),
						NamaMK: input["namaMk"].(string),
						SKS:    input["sks"].(int),
					}
					mataKuliahLock.Lock()
					mataKuliahStore[newMk.ID] = newMk
					mataKuliahLock.Unlock()
					return newMk, nil
				},
			},
			"daftarkanMataKuliahUntukMahasiswa": &graphql.Field{
				Type: mahasiswaType, // Mengembalikan mahasiswa yang diupdate
				Args: graphql.FieldConfigArgument{
					"mahasiswaId":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"mataKuliahId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					mhsID, _ := p.Args["mahasiswaId"].(string)
					mkID, _ := p.Args["mataKuliahId"].(string)

					mahasiswaLock.Lock()
					defer mahasiswaLock.Unlock()
					mataKuliahLock.RLock() // Hanya perlu read lock untuk cek exist mk
					defer mataKuliahLock.RUnlock()

					mhs, mhsExists := mahasiswaStore[mhsID]
					if !mhsExists {
						return nil, fmt.Errorf("Mahasiswa dengan ID %s tidak ditemukan", mhsID)
					}

					_, mkExists := mataKuliahStore[mkID]
					if !mkExists {
						return nil, fmt.Errorf("Mata Kuliah dengan ID %s tidak ditemukan", mkID)
					}

					// Cek duplikasi
					for _, existingMkID := range mhs.MataKuliahID {
						if existingMkID == mkID {
							return mhs, fmt.Errorf("Mahasiswa sudah mengambil mata kuliah ini") // Atau return mhs saja tanpa error jika dianggap idempotent
						}
					}

					mhs.MataKuliahID = append(mhs.MataKuliahID, mkID)
					mahasiswaStore[mhsID] = mhs
					return mhs, nil
				},
			},
		},
	})

	var err error
	schema, err = graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
	if err != nil {
		log.Fatalf("Gagal membuat skema GraphQL: %v", err)
	}
}

// --- HTTP Handler ---
func graphqlHandler(w http.ResponseWriter, r *http.Request) {
	// Pastikan skema sudah di-build
	if schema.QueryType() == nil { // Cara sederhana untuk cek apakah buildSchema sudah dipanggil
		buildSchema()
	}

	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	result := graphql.Do(graphql.Params{
		Schema:         schema,
		RequestString:  params.Query,
		VariableValues: params.Variables,
		OperationName:  params.OperationName,
	})

	if len(result.Errors) > 0 {
		log.Printf("GraphQL errors: %v", result.Errors)
		// Anda mungkin ingin mengembalikan status 200 OK dengan error di body,
		// atau status error HTTP tergantung preferensi.
		// Umumnya GraphQL mengembalikan 200 OK dengan error di payload.
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func main() {
	initData()    // Inisialisasi data contoh
	buildSchema() // Build skema saat aplikasi dimulai

	// Menyajikan file statis dari direktori "static" untuk path "/static/"
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/graphql", graphqlHandler)

	// Handler untuk halaman utama yang menyajikan GraphiQL
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `
            <!DOCTYPE html>
            <html>
                <head>
                    <title>Simple GraphiQL Example</title>
                    <link href="/static/graphiql.min.css" rel="stylesheet" />
                </head>
                <body style="margin: 0;">
                    <div id="graphiql" style="height: 100vh;"></div>

                    <script crossorigin src="/static/react.production.min.js"></script>
                    <script crossorigin src="/static/react-dom.production.min.js"></script>
                    <script crossorigin src="/static/graphiql.min.js"></script>

                    <script>
                        const fetcher = GraphiQL.createFetcher({url: '/graphql'});
                        ReactDOM.render(
                            React.createElement(GraphiQL, { fetcher: fetcher }),
                            document.getElementById('graphiql'),
                        );
                    </script>
                </body>
            </html>
        `)
	})

	log.Println("Server GraphQL berjalan di http://localhost:8081")
	log.Println("GraphiQL tersedia di http://localhost:8081/")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}

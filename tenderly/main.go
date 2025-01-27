package tenderly

import (
	"fmt"
	"github.com/linxGnu/grocksdb"
	"log"
)

func main() {
	// Kreiranje opcija za RocksDB
	options := grocksdb.NewDefaultOptions()
	defer options.Destroy()
	options.SetCreateIfMissing(true)

	// Otvaranje RocksDB baze
	db, err := grocksdb.OpenDb(options, "testdb")
	if err != nil {
		log.Fatalf("Greška prilikom otvaranja baze: %v", err)
	}
	defer db.Close()

	// Kreiranje write opcija
	writeOptions := grocksdb.NewDefaultWriteOptions()
	defer writeOptions.Destroy()

	// Pisanje ključeva u bazu
	err = db.Put(writeOptions, []byte("key1"), []byte("value1"))
	if err != nil {
		log.Fatalf("Greška prilikom pisanja u bazu: %v", err)
	}
	err = db.Put(writeOptions, []byte("key2"), []byte("value2"))
	if err != nil {
		log.Fatalf("Greška prilikom pisanja u bazu: %v", err)
	}

	fmt.Println("Uspešno zapisano u bazu.")

	// Kreiranje read opcija
	readOptions := grocksdb.NewDefaultReadOptions()
	defer readOptions.Destroy()

	// Čitanje vrednosti iz baze
	value1, err := db.Get(readOptions, []byte("key1"))
	if err != nil {
		log.Fatalf("Greška prilikom čitanja iz baze: %v", err)
	}
	defer value1.Free()
	fmt.Printf("Key1: %s\n", value1.Data())

	value2, err := db.Get(readOptions, []byte("key2"))
	if err != nil {
		log.Fatalf("Greška prilikom čitanja iz baze: %v", err)
	}
	defer value2.Free()
	fmt.Printf("Key2: %s\n", value2.Data())
}

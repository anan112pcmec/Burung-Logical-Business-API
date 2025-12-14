package maintain_cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gorm.io/gorm"

	data_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/data"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

func DataAlamatEkspedisiUp(db *gorm.DB) {
	var idsAlamatEks []int64
	if err := db.Model(&models.AlamatEkspedisi{}).Pluck("id", &idsAlamatEks).Error; err != nil {
		return
	}

	var alamatEkspedisiData []models.AlamatEkspedisi
	if err := db.Where("id IN ?", idsAlamatEks).Find(&alamatEkspedisiData).Error; err != nil {
		return
	}

	if data_cache.DataAlamatEkspedisi == nil {
		data_cache.DataAlamatEkspedisi = make(map[string]map[int64]models.AlamatEkspedisi, len(alamatEkspedisiData))
	}

	for i := range alamatEkspedisiData {
		kota := alamatEkspedisiData[i].Kota
		if _, ok := data_cache.DataAlamatEkspedisi[kota]; !ok {
			data_cache.DataAlamatEkspedisi[kota] = make(map[int64]models.AlamatEkspedisi)
		}
		data_cache.DataAlamatEkspedisi[kota][alamatEkspedisiData[i].ID] = alamatEkspedisiData[i]
	}
}

func DataAlamatEkspedisiChange(DataEkspedisi models.AlamatEkspedisi, kota string, id int64, db *gorm.DB) {
	data_cache.DataAlamatEkspedisi[kota][id] = DataEkspedisi
}

func DataOperasionalPengirimanUp() {
	log.Println("[TRACE] Memulai update OperationalPengirimanData")

	// Set path dan nama file
	filePath := "C:/Burung_App/Project_Source/Backend-1/app/operational_data/pengiriman_burung.json"
	log.Printf("[TRACE] Membaca file JSON dari path: %s", filePath)

	// Buka file JSON
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] Gagal membuka file JSON: %v", err)
		return
	}
	defer file.Close()

	// Decode JSON ke struct global
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data_cache.OperationalPengirimanData); err != nil {
		log.Printf("[ERROR] Gagal decode JSON ke struct: %v", err)
		return
	}
	log.Println("[TRACE] JSON berhasil didecode ke OperationalPengirimanData")

	// Update tarif pengiriman utama
	data_cache.DataTarifPengiriman.TarifKurirPerKm = data_cache.OperationalPengirimanData.CommittedOperationalData.DataTarifPengiriman.TarifKurirPerKm
	data_cache.DataTarifPengiriman.TarifKurirPerKg = data_cache.OperationalPengirimanData.CommittedOperationalData.DataTarifPengiriman.TarifKurirPerKg
	data_cache.DataTarifPengiriman.TarifSistem = data_cache.OperationalPengirimanData.CommittedOperationalData.DataTarifPengiriman.TarifSistem
	log.Printf("[TRACE] DataTarifPengiriman updated: %+v", data_cache.DataTarifPengiriman)

	// Update DataTarifJenisPengiriman
	data_cache.DataTarifJenisPengiriman = make(map[string]data_cache.JenisPengiriman, len(data_cache.OperationalPengirimanData.CommittedOperationalData.DataJenisPengiriman))
	for _, jenis := range data_cache.OperationalPengirimanData.CommittedOperationalData.DataJenisPengiriman {
		data_cache.DataTarifJenisPengiriman[jenis.Nama] = jenis
		log.Printf("[TRACE] JenisPengiriman ditambahkan: %s -> %+v", jenis.Nama, jenis)
	}

	fmt.Println("Harganyo", data_cache.DataTarifJenisPengiriman["reguler"].Harga)

	log.Println("[TRACE] OperationalPengirimanData berhasil di-update dari JSON")
}

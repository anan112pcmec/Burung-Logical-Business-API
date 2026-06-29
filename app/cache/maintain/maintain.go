package maintain_cache

import (
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

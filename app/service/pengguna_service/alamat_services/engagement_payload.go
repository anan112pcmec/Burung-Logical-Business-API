package pengguna_alamat_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"

)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Masukan Alamat Pengguna
// ///////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMasukanAlamatPengguna struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	PanggilanAlamat   string                             `json:"panggilan_alamat"`
	NomorTelephone    string                             `json:"nomor_telefon"`
	NamaAlamat        string                             `json:"nama_alamat"`
	Provinsi          string                             `json:"provinsi"`
	Kota              string                             `json:"kota"`
	KodePos           string                             `json:"kode_pos"`
	KodeNegara        string                             `json:"kode_negara"`
	Deskripsi         string                             `json:"deskripsi"`
	Longitude         float64                            `json:"longitude"`
	Latitude          float64                            `json:"latitude"`
}

// ///////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Edit Alamat Pengguna
// ///////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditAlamatPengguna struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdAlamatPengguna  int64                              `json:"id_alamat_pengguna"`
	PanggilanAlamat   string                             `json:"panggilan_alamat"`
	NomorTelephone    string                             `json:"nomor_telefon"`
	NamaAlamat        string                             `json:"nama_alamat"`
	Provinsi          string                             `json:"provinsi"`
	Kota              string                             `json:"kota"`
	KodePos           string                             `json:"kode_pos"`
	KodeNegara        string                             `json:"kode_negara"`
	Deskripsi         string                             `json:"deskripsi"`
	Longitude         float64                            `json:"longitude"`
	Latitude          float64                            `json:"latitude"`
}

// ///////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Hapus Alamat Pengguna
// ///////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusAlamatPengguna struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdAlamatPengguna  int64                              `json:"id_alamat_hapus_alamat"`
}

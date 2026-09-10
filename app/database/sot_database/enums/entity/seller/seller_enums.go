package seller_enums

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SELLER JENIS ENUM
// 3 ENUM UNTUK TABLE SELLER
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	Brands       string = "Brands"
	Distributors string = "Distributors"
	Personal     string = "Personal"
)

func NamaJenisSellerEnums() string {
	return "jenis_seller"
}

func JenisSellerEnums() []string {
	return []string{Brands, Distributors, Personal}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SELLER DEDICATION ENUM
// 17 ENUM UNTUK TABLE SELLER
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	PakaianFashion     string = "Pakaian & Fashion"
	KosmetikKecantikan string = "Kosmetik & Kecantikan"
	ElektronikGadget   string = "Elektronik & Gadget"
	BukuMedia          string = "Buku & Media"
	MakananMinuman     string = "Makanan & Minuman"
	IbuBayi            string = "Ibu & Bayi"
	Mainan             string = "Mainan"
	OlahragaOutdoor    string = "Olahraga & Outdoor"
	OtomotifSparepart  string = "Otomotif & Sparepart"
	RumahTangga        string = "Rumah Tangga"
	AlatTulis          string = "Alat Tulis"
	PerhiasanAksesoris string = "Perhiasan & Aksesoris"
	ProdukDigital      string = "Produk Digital"
	BangunanPerkakas   string = "Bangunan & Perkakas"
	MusikInstrumen     string = "Musik & Instrumen"
	FilmBroadcasting   string = "Film & Broadcasting"
	SemuaBarang        string = "Semua Barang"
)

func NamaSellerDedicationEnum() string {
	return "seller_dedication"
}

func SellerDedicationEnums() []string {
	return []string{PakaianFashion, KosmetikKecantikan, ElektronikGadget, BukuMedia, MakananMinuman, IbuBayi, Mainan, OlahragaOutdoor, OtomotifSparepart, RumahTangga, PerhiasanAksesoris, ProdukDigital, BangunanPerkakas, MusikInstrumen, SemuaBarang}
}

const (
	Pending   string = "Pending"
	Confirmed string = "Confirmed"
	Declined  string = "Declined"
)

func NamaStatusJenisSellerEnums() string {
	return "status_jenis_seller"
}

func StatusJenisSellerEnums() []string {
	return []string{Pending, Confirmed, Declined}
}

const (
	Draft   string = "Draft"
	Aktif   string = "Aktif"
	Selesai string = "Selesai"
)

func NamaStatusDiskonProdukEnums() string {
	return "status_diskon_produk"
}

func StatusDiskonProdukEnums() []string {
	return []string{Draft, Aktif, Selesai}
}

const (
	Waiting string = "Waiting"
	Applied string = "Applied"
)

func NamaStatusBarangDiDiskonEnums() string {
	return "status_barang_di_diskon"
}

func StatusBarangDiDiskonEnums() []string {
	return []string{Waiting, Applied}
}

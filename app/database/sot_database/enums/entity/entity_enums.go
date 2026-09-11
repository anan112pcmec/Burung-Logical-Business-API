package entity_enums

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// STATUS ENUM
// 2 ENUM UNTUK TABLE PENGGUNA
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	Online  string = "Online"
	Offline string = "Offline"
)

const (
	Pengguna string = "pengguna"
	Seller   string = "seller"
	Kurir    string = "kurir"
)

func NamaEntityJenisEnums() string {
	return "jenis_entity"
}

func EntityJenisEnums() []string {
	return []string{Pengguna, Seller, Kurir}
}

func NamaEntityStatusEnums() string {
	return "status"
}

func EntityStatusEnums() []string {
	return []string{Online, Offline}
}

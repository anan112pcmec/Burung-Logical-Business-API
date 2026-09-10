package barang_enums

const (
	Pending  string = "Pending"
	Ready    string = "Ready"
	Dipesan  string = "Dipesan"
	Diproses string = "Diproses"
	Terjual  string = "Terjual"
	Down     string = "Down"
)

func NamaStatusVarianBarangEnums() string {
	return "status_varian"
}

func StatusVarianBarangEnums() []string {
	return []string{Pending, Ready, Dipesan, Diproses, Terjual, Down}
}

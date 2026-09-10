package keranjang_enums

const (
	Ready   string = "Ready"
	UnReady string = "UnReady"
)

func NamaStatusKeranjangEnums() string {
	return "status_keranjang"
}

func StatusKeranjangEnums() []string {
	return []string{Ready, UnReady}
}

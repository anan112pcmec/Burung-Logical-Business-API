package pengiriman_enums

// Enums jenis pengiriman
const (
	Reguler = "Reguler"
	Express = "Express"
	Instant = "Instant"
)

func NamaJenisLayananKurirEnums() string {
	return "jenis_layanan_kurir"
}

func JenisLayananKurirEnums() []string {
	return []string{Reguler, Express, Instant}
}

// Enums untuk pengiriman non ekspedisi

const (
	// untuk pengiriman non ekspedisi
	Waiting      = "Waiting"
	PickedUp     = "Picked Up"
	Diperjalanan = "Diperjalanan"
	Sampai       = "Sampai"
	Trouble      = "Trouble"

	// Enums untuk pengiriman ekspedisi

	DikirimEkspedisi           = "Dikirim"
	SampaiAgentEkspedisi       = "Sampai Agent"
	MasukGateaway              = "Masuk Gateway"
	SampaiAgentTujuanEkspedisi = "Sampai Agent Tujuan"
	DikirimAgentEkspedisi      = "Dikirim Agent"
)

func NamaStatusPengirimanNonEkspedisi() string {
	return "status_pengiriman"
}

func StatusPengirimanNonEkspedisi() []string {
	return []string{Waiting, PickedUp, Diperjalanan, Sampai, Trouble}
}

func NamaStatusPengirimanEkspedisiEnums() string {
	return "status_pengiriman_ekspedisi"
}

func StatusPengirimanEkspedisiEnums() []string {
	return []string{PickedUp, Waiting, DikirimEkspedisi, SampaiAgentEkspedisi, MasukGateaway, DikirimAgentEkspedisi, Sampai}
}

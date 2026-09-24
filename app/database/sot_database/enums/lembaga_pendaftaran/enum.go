package lembaga_pendaftaran

const (
	// Indonesia
	DJKI string = "DJKI"

	// Internasional / Global
	WIPO string = "WIPO"

	// Regional
	EUIPO string = "EUIPO"
	EPO   string = "EPO"
	ARIPO string = "ARIPO"
	OAPI  string = "OAPI"

	// Nasional Negara Lain
	USPTO       string = "USPTO"        // Amerika Serikat
	CNIPA       string = "CNIPA"        // Tiongkok
	JPO         string = "JPO"          // Jepang
	KIPO        string = "KIPO"         // Korea Selatan
	UKIPO       string = "UKIPO"        // Inggris
	IPAustralia string = "IP Australia" // Australia
	CIPO        string = "CIPO"         // Kanada
	INPI        string = "INPI"         // Prancis / Brasil
	DPMA        string = "DPMA"         // Jerman
	IMPI        string = "IMPI"         // Meksiko
	IPOS        string = "IPOS"         // Singapura
	MyIPO       string = "MyIPO"        // Malaysia
	DIP         string = "DIP"          // Thailand
	IPPH        string = "IPPH"         // Filipina
)

// DaftarLembagaValid adalah map untuk validasi lembaga pendaftaran
var DaftarLembagaValid map[string]string = map[string]string{
	"DJKI":         DJKI,
	"WIPO":         WIPO,
	"EUIPO":        EUIPO,
	"EPO":          EPO,
	"ARIPO":        ARIPO,
	"OAPI":         OAPI,
	"USPTO":        USPTO,
	"CNIPA":        CNIPA,
	"JPO":          JPO,
	"KIPO":         KIPO,
	"UKIPO":        UKIPO,
	"IP Australia": IPAustralia,
	"CIPO":         CIPO,
	"INPI":         INPI,
	"DPMA":         DPMA,
	"IMPI":         IMPI,
	"IPOS":         IPOS,
	"MyIPO":        MyIPO,
	"DIP":          DIP,
	"IPPH":         IPPH,
}

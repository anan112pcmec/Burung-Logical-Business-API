package payment_out_constanta

import (
	"encoding/base64"
	"os"

)

var (
	SecretKey string = os.Getenv("FLIP_DISBURSTMENT_SECRET_KEY")
	Token     string = os.Getenv("FLIP_DISBURSTMENT_TOKEN")
)
var Auth = ("Basic " + base64.StdEncoding.EncodeToString([]byte(SecretKey+":")))

var CityFlipJawaCode = map[string]string{
	// Jawa Barat
	"bekasi":        "102",
	"purwakarta":    "103",
	"karawang":      "106",
	"bogor":         "108",
	"sukabumi":      "109",
	"cianjur":       "110",
	"bandung":       "111",
	"sumedang":      "112",
	"tasikmalaya":   "113",
	"garut":         "114",
	"ciamis":        "115",
	"cirebon":       "116",
	"kuningan":      "117",
	"indramayu":     "118",
	"majalengka":    "119",
	"subang":        "121",
	"bandung barat": "122",
	"banjar":        "180",
	"lainnya":       "188",

	// Banten
	"lebak":             "201",
	"pandeglang":        "202",
	"serang":            "203",
	"tangerang":         "204",
	"cilegon":           "291",
	"tangerang selatan": "294",

	// DKI Jakarta
	"jakarta pusat":    "391",
	"jakarta utara":    "392",
	"jakarta barat":    "393",
	"jakarta selatan":  "394",
	"jakarta timur":    "395",
	"kepulauan seribu": "396",

	// DI Yogyakarta
	"bantul":           "501",
	"sleman":           "502",
	"gunung kidul":     "503",
	"kulon progo":      "504",
	"yogyakarta":       "591",
	"lainnya di yogya": "588",

	// Jawa Tengah
	"semarang":               "901",
	"kendal":                 "902",
	"demak":                  "903",
	"grobogan":               "904",
	"pekalongan":             "905",
	"tegal":                  "906",
	"brebes":                 "907",
	"pati":                   "908",
	"kudus":                  "909",
	"pemalang":               "910",
	"jepara":                 "911",
	"rembang":                "912",
	"blora":                  "913",
	"banyumas":               "914",
	"cilacap":                "915",
	"purbalingga":            "916",
	"banjarnegara":           "917",
	"magelang":               "918",
	"temanggung":             "919",
	"wonosobo":               "920",
	"purworejo":              "921",
	"kebumen":                "922",
	"klaten":                 "923",
	"boyolali":               "924",
	"sragen":                 "925",
	"sukoharjo":              "926",
	"karanganyar":            "927",
	"wonogiri":               "928",
	"batang":                 "929",
	"salatiga":               "992",
	"surakarta/solo":         "996",
	"lainnya di jawa tengah": "988",

	// Jawa Timur
	"gresik":                "1201",
	"sidoarjo":              "1202",
	"mojokerto":             "1203",
	"jombang":               "1204",
	"sampang":               "1205",
	"pamekasan":             "1206",
	"sumenep":               "1207",
	"bangkalan":             "1208",
	"bondowoso":             "1209",
	"banyuwangi":            "1211",
	"jember":                "1212",
	"malang":                "1213",
	"pasuruan":              "1214",
	"probolinggo":           "1215",
	"lumajang":              "1216",
	"kediri":                "1217",
	"nganjuk":               "1218",
	"tulungagung":           "1219",
	"trenggalek":            "1220",
	"blitar":                "1221",
	"madiun":                "1222",
	"ngawi":                 "1223",
	"magetan":               "1224",
	"ponorogo":              "1225",
	"pacitan":               "1226",
	"bojonegoro":            "1227",
	"tuban":                 "1228",
	"lamongan":              "1229",
	"situbondo":             "1230",
	"surabaya":              "1291",
	"batu":                  "1271",
	"lainnya di jawa timur": "1288",
}

const (
	Pending  string = "PENDING"
	Done     string = "DONE"
	Canceled string = "CANCELLED"
)

func StatusDoneDisbursment(s string) bool {
	return s == Done
}

func StatusCanceledDisbursment(s string) bool {
	return s == Canceled
}

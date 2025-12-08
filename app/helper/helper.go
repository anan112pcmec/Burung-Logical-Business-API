package helper

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
	mrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
)

func DecodeJSONBody(r *http.Request, dst interface{}) error {
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	return nil
}

// GenerateRandomDigits menghasilkan angka random dengan panjang 3-5 digit
func GenerateRandomDigits() string {
	rand.Seed(time.Now().UnixNano())

	length := rand.Intn(3) + 3

	min := intPow(10, length-1)   // contoh: 100 untuk 3 digit
	max := intPow(10, length) - 1 // contoh: 999 untuk 3 digit

	return strconv.Itoa(rand.Intn(max-min+1) + min)
}

func intPow(base, exp int) int {
	result := 1
	for exp > 0 {
		result *= base
		exp--
	}
	return result
}

func ConvertJenisBarang(jenis string) string {
	// Map internal -> DB
	mapJenis := map[string]string{
		"Pakaian&Fashion":     "Pakaian & Fashion",
		"Kosmetik&Kecantikan": "Kosmetik & Kecantikan",
		"Elektronik&Gadget":   "Elektronik & Gadget",
		"Buku&Media":          "Buku & Media",
		"Makanan&Minuman":     "Makanan & Minuman",
		"Ibu&Bayi":            "Ibu & Bayi",
		"Mainan":              "Mainan",
		"Olahraga&Outdoor":    "Olahraga & Outdoor",
		"Otomotif&Sparepart":  "Otomotif & Sparepart",
		"RumahTangga":         "Rumah Tangga",
		"AlatTulis":           "Alat Tulis",
		"Perhiasan&Aksesoris": "Perhiasan & Aksesoris",
		"ProdukDigital":       "ProdukDigital",
		"Bangunan&Perkakas":   "Bangunan & Perkakas",
		"Musik&Instrumen":     "Musik & Instrumen",
		"Film&Broadcasting":   "Film & Broadcasting",
		"SemuaBarang":         "Semua Barang",
	}

	if val, ok := mapJenis[jenis]; ok {
		return val
	}
	// fallback kalau tidak ada mapping
	return jenis
}

func ConvertJenisBarangReverse(jenis string) string {
	// Map DB -> internal
	mapReverse := map[string]string{
		"Pakaian & Fashion":     "Pakaian&Fashion",
		"Kosmetik & Kecantikan": "Kosmetik&Kecantikan",
		"Elektronik & Gadget":   "Elektronik&Gadget",
		"Buku & Media":          "Buku&Media",
		"Makanan & Minuman":     "Makanan&Minuman",
		"Ibu & Bayi":            "Ibu&Bayi",
		"Mainan":                "Mainan",
		"Olahraga & Outdoor":    "Olahraga&Outdoor",
		"Otomotif & Sparepart":  "Otomotif&Sparepart",
		"Rumah Tangga":          "RumahTangga",
		"Alat Tulis":            "AlatTulis",
		"Perhiasan & Aksesoris": "Perhiasan&Aksesoris",
		"ProdukDigital":         "ProdukDigital",
		"Bangunan & Perkakas":   "Bangunan&Perkakas",
		"Musik & Instrumen":     "Musik&Instrumen",
		"Film & Broadcasting":   "Film&Broadcasting",
		"Semua Barang":          "SemuaBarang",
	}

	if val, ok := mapReverse[jenis]; ok {
		return val
	}
	// fallback kalau tidak ada mapping
	return jenis
}

func randomString(length int, charset string) string {
	if length <= 0 || len(charset) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(length)

	lenCharset := len(charset)
	maxAcceptable := 255 - (255 % lenCharset)

	i := 0
	for i < length {
		var b [1]byte
		if _, err := crand.Read(b[:]); err != nil {
			// fallback: gunakan math/rand jika crypto/rand error
			r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
			for ; i < length; i++ {
				sb.WriteByte(charset[r.Intn(lenCharset)])
			}
			return sb.String()
		}

		if int(b[0]) >= maxAcceptable {
			// buang hasil supaya tidak bias
			continue
		}

		idx := int(b[0]) % lenCharset
		sb.WriteByte(charset[idx])
		i++
	}

	return sb.String()
}

func GenerateAutoPaymentId(db *gorm.DB) (string, error) {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	alphanum := letters + numbers

	part1 := randomString(4, letters)  // 4 huruf
	part2 := randomString(5, numbers)  // 5 angka
	part3 := randomString(6, alphanum) // 6 huruf/angka

	final := part1 + "-" + part2 + "-" + part3

	var id_data_transaksi []int64
	err_kode := db.Model(models.Transaksi{}).Select("id").Where(models.Transaksi{KodeOrderSistem: final}).Limit(2).Scan(&id_data_transaksi).Error
	if err_kode != nil {
		return final, err_kode
	}

	if len(id_data_transaksi) != 0 {
		return "", err_kode
	}

	return final, nil
}

func Hitungtotal(input []int64) int64 {
	var total int64 = 0
	for _, biaya := range input {
		total += biaya
	}
	return total
}

func UpdateSocialMediaDispatch(data models.EntitySocialMedia) []string {
	var hasil []string
	if data.Instagram != "" {
		hasil = append(hasil, "instagram")
	}

	if data.Whatsapp != "" {
		hasil = append(hasil, "whatsapp")
	}

	if data.Facebook != "" {
		hasil = append(hasil, "facebook")
	}

	if data.TikTok != "" {
		hasil = append(hasil, "tiktok")

	}

	return hasil
}

func GenerateOTP() string {
	otp := ""
	for i := 0; i < 8; i++ {
		// ambil angka random 0–9
		n, _ := crand.Int(crand.Reader, big.NewInt(10))
		otp += fmt.Sprintf("%d", n.Int64())
	}
	return otp
}

func SanitasiKoordinat(Latitude *float64, Longitude *float64) {
	if Latitude == nil || Longitude == nil {
		return
	}

	lat := *Latitude
	long := *Longitude

	// 🔥 JIKA KELEBIHAN DIGIT (contoh: 106.8299 → 10.68299)
	if math.Abs(lat) > 90 {
		lat = lat / 10
	}
	if math.Abs(long) > 180 {
		long = long / 10
	}

	// 🔥 PASTIKAN DALAM RANGE VALID BUMI
	if lat > 90 {
		lat = 90
	}
	if lat < -90 {
		lat = -90
	}
	if long > 180 {
		long = 180
	}
	if long < -180 {
		long = -180
	}

	// 🚀 TULIS BALIK HASILNYA
	*Latitude = lat
	*Longitude = long
}

func ParseCoordinates(koordinatMentah string) (float64, float64, error) {
	parts := strings.Split(strings.TrimSpace(koordinatMentah), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("format koordinat salah: %s", koordinatMentah)
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("gagal parse latitude: %v", err)
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("gagal parse longitude: %v", err)
	}

	// Validasi range
	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("latitude tidak valid: %f", lat)
	}
	if lng < -180 || lng > 180 {
		return 0, 0, fmt.Errorf("longitude tidak valid: %f", lng)
	}

	// Fungsi bulatkan maksimal 6 desimal
	round6 := func(f float64) float64 {
		return math.Round(f*1_000_000) / 1_000_000
	}

	lat = round6(lat)
	lng = round6(lng)

	return lat, lng, nil
}

func GenerateIdempotencyKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func GenerateMediaKeyPhoto() string {
	b := make([]byte, 12)
	mrand.Read(b)
	return fmt.Sprintf("%s-pto", hex.EncodeToString(b))
}

func GenerateMediaKeyVideo() string {
	b := make([]byte, 9)
	mrand.Read(b)
	return fmt.Sprintf("%s-vd", hex.EncodeToString(b))
}

func GenerateMediaKeyDokumen() string {
	b := make([]byte, 11)
	mrand.Read(b)
	return fmt.Sprintf("%s-dok", hex.EncodeToString(b))
}

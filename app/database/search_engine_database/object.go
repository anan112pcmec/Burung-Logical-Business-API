package search_engine

type BarangInduk struct {
	IDKey            int64  `json:"id"`
	IdBarang         int64  `json:"id_barang_induk"`
	NamaBarang       string `json:"nama_barang_induk"`
	IdSeller         int32  `json:"id_seller_barang_induk"`
	OriginalKategori string `json:"original_kategori"`
	Deskripsi        string `json:"deskripsi"`
	Jenis            string `json:"jenis_barang_induk"`
	TGL              string `json:"tanggal_rilis_barang_induk"`
	Viewed           int32  `json:"viewed_barang_induk"`
	Likes            int32  `json:"likes_barang_induk"`
}

type Seller struct {
	IDKey            int32  `json:"id"`
	IdSeller         int32  `json:"id_seller"`
	NamaSeller       string `json:"nama_seller"`
	JenisSeller      string `json:"jenis_seller"`
	SellerDedication string `json:"seller_dedication_seller"`
	FollowerTotal    int32  `json:"follower_total_seller"`
}

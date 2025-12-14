-- Create ENUM types first
CREATE TYPE status AS ENUM ('Online', 'Offline');
CREATE TYPE jenis_seller AS ENUM ('Personal', 'Bisnis');
CREATE TYPE seller_dedication AS ENUM ('Semua Barang', 'Elektronik', 'Fashion', 'Makanan', 'Lainnya');
CREATE TYPE jenis_layanan_kurir AS ENUM ('Reguler', 'Express', 'Same Day', 'Instant');
CREATE TYPE status_kurir AS ENUM ('On', 'Off');
CREATE TYPE jenis_kendaraan_kurir AS ENUM ('Motor', 'Mobil', 'Sepeda', 'Jalan Kaki', 'Unknown');

-- Table: pengguna
CREATE TABLE pengguna (
    id_user BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL DEFAULT '',
    nama TEXT NOT NULL DEFAULT '',
    email VARCHAR(100) NOT NULL DEFAULT '',
    password_hash VARCHAR(250) NOT NULL DEFAULT '',
    pin_hash VARCHAR(250) NOT NULL DEFAULT '',
    status status NOT NULL DEFAULT 'Offline',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT unique_email_pengguna UNIQUE (email)
);

-- Create index for soft delete
CREATE INDEX idx_pengguna_deleted_at ON pengguna(deleted_at);

-- Table: seller
CREATE TABLE seller (
    id_seller SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL DEFAULT '',
    nama VARCHAR(150) NOT NULL DEFAULT '',
    email VARCHAR(150) NOT NULL DEFAULT '',
    jenis jenis_seller NOT NULL DEFAULT 'Personal',
    seller_dedication seller_dedication NOT NULL DEFAULT 'Semua Barang',
    jam_operasional TEXT NOT NULL DEFAULT '',
    punchline TEXT NOT NULL DEFAULT '',
    password_hash VARCHAR(250) NOT NULL DEFAULT '',
    deskripsi TEXT NOT NULL DEFAULT '',
    follower_total INT NOT NULL DEFAULT 0,
    status status NOT NULL DEFAULT 'Offline',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- Create index for soft delete
CREATE INDEX idx_seller_deleted_at ON seller(deleted_at);

-- Table: kurir
CREATE TABLE kurir (
    id_kurir BIGSERIAL PRIMARY KEY,
    nama VARCHAR(150) NOT NULL DEFAULT '',
    username TEXT NOT NULL,
    email VARCHAR(150) NOT NULL DEFAULT '',
    jenis jenis_layanan_kurir NOT NULL DEFAULT 'Reguler',
    password_hash VARCHAR(250) NOT NULL DEFAULT '',
    deskripsi TEXT NOT NULL DEFAULT '',
    status status NOT NULL DEFAULT 'Offline',
    status_bid status_kurir NOT NULL DEFAULT 'Off',
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    jumlah_pengiriman INT NOT NULL DEFAULT 0,
    rating REAL DEFAULT 0,
    jumlah_rating INT DEFAULT 0,
    jenis_kendaraan jenis_kendaraan_kurir DEFAULT 'Unknown',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- Create index for soft delete
CREATE INDEX idx_kurir_deleted_at ON kurir(deleted_at);

-- Optional: Create trigger for auto-updating updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to all tables
CREATE TRIGGER update_pengguna_updated_at BEFORE UPDATE ON pengguna
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_seller_updated_at BEFORE UPDATE ON seller
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_kurir_updated_at BEFORE UPDATE ON kurir
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================================================================================
-- ENGAGEMENT & SOCIAL MEDIA TABLES
-- ==================================================================================

-- Additional ENUM types for engagement tables
CREATE TYPE status_keranjang AS ENUM ('Active', 'Checkout', 'Expired');
CREATE TYPE status_jenis_seller AS ENUM ('Pending', 'Approved', 'Rejected');
CREATE TYPE status_perizinan AS ENUM ('Pending', 'Approved', 'Rejected');
CREATE TYPE roda_kendaraan_kurir AS ENUM ('2', '3', '4', '6');
CREATE TYPE mode_bid_kurir AS ENUM ('Area', 'Point');
CREATE TYPE status_barang_di_diskon AS ENUM ('Waiting', 'Active', 'Expired');

-- Table: entity_social_media
CREATE TABLE entity_social_media (
    id_social_media BIGSERIAL PRIMARY KEY,
    entity_id BIGINT NOT NULL,
    whatsapp VARCHAR(20),
    facebook TEXT,
    tiktok TEXT,
    instagram TEXT,
    metadata BYTEA,
    entity_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_entity_social_media_deleted_at ON entity_social_media(deleted_at);
CREATE INDEX idx_entity_social_media_entity ON entity_social_media(entity_id, entity_type);

-- Table: komentar
CREATE TABLE komentar (
    id_komentar BIGSERIAL PRIMARY KEY,
    id_barang_induk INT NOT NULL,
    id_entity BIGINT NOT NULL,
    jenis_entity VARCHAR(50) NOT NULL,
    komentar TEXT NOT NULL,
    is_seller BOOLEAN NOT NULL DEFAULT FALSE,
    dibalas BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_komentar_deleted_at ON komentar(deleted_at);
CREATE INDEX idx_komentar_barang ON komentar(id_barang_induk);

-- Table: komentar_child
CREATE TABLE komentar_child (
    id_komentar_child BIGSERIAL PRIMARY KEY,
    id_komentar BIGINT NOT NULL,
    id_entity BIGINT NOT NULL,
    jenis_entity VARCHAR(50) NOT NULL,
    komentar TEXT NOT NULL,
    is_seller BOOLEAN NOT NULL DEFAULT FALSE,
    mention TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_komentar_child_komentar FOREIGN KEY (id_komentar) 
        REFERENCES komentar(id_komentar) ON DELETE CASCADE
);

CREATE INDEX idx_komentar_child_deleted_at ON komentar_child(deleted_at);
CREATE INDEX idx_komentar_child_parent ON komentar_child(id_komentar);

-- ==================================================================================
-- ENGAGEMENT PENGGUNA (USER ENGAGEMENT)
-- ==================================================================================

-- Table: keranjang
CREATE TABLE keranjang (
    id_keranjang BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_seller INT NOT NULL,
    id_barang_induk INT NOT NULL,
    id_kategori_barang BIGINT NOT NULL,
    jumlah SMALLINT NOT NULL,
    status status_keranjang NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_keranjang_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user),
    CONSTRAINT fk_keranjang_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller)
);

CREATE INDEX idx_keranjang_deleted_at ON keranjang(deleted_at);
CREATE INDEX idx_keranjang_pengguna ON keranjang(id_pengguna);

-- Table: barang_disukai
CREATE TABLE barang_disukai (
    id_barang_disukai BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_barang_induk INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_barang_disukai_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user)
);

CREATE INDEX idx_barang_disukai_deleted_at ON barang_disukai(deleted_at);
CREATE INDEX idx_barang_disukai_pengguna ON barang_disukai(id_pengguna);

-- Table: wishlist
CREATE TABLE wishlist (
    id_wishlist BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_barang_induk INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_wishlist_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user)
);

CREATE INDEX idx_wishlist_deleted_at ON wishlist(deleted_at);
CREATE INDEX idx_wishlist_pengguna ON wishlist(id_pengguna);

-- Table: alamat_pengguna
CREATE TABLE alamat_pengguna (
    id_alamat_user BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    panggilan_alamat VARCHAR(250) NOT NULL,
    nomor_telefon VARCHAR(20) NOT NULL,
    nama_alamat TEXT NOT NULL,
    provinsi VARCHAR(100) NOT NULL,
    kota VARCHAR(100) NOT NULL,
    kode_pos VARCHAR(40) NOT NULL,
    kode_negara VARCHAR(10) NOT NULL DEFAULT 'IDN',
    deskripsi TEXT,
    longitude DECIMAL(11,8),
    latitude DECIMAL(11,8),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_alamat_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user)
);

CREATE INDEX idx_alamat_pengguna_deleted_at ON alamat_pengguna(deleted_at);
CREATE INDEX idx_alamat_pengguna_user ON alamat_pengguna(id_pengguna);

-- Table: review
CREATE TABLE review (
    id_review BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_barang_induk INT NOT NULL,
    rating NUMERIC(2,1) NOT NULL,
    ulasan TEXT,
    "like" BIGINT DEFAULT 0,
    dislike BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_review_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user)
);

CREATE INDEX idx_review_deleted_at ON review(deleted_at);
CREATE INDEX idx_review_pengguna ON review(id_pengguna);
CREATE INDEX idx_review_barang ON review(id_barang_induk);

-- Table: review_like
CREATE TABLE review_like (
    id_like_review BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_review BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_review_like_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user),
    CONSTRAINT fk_review_like_review FOREIGN KEY (id_review) REFERENCES review(id_review)
);

CREATE INDEX idx_review_like_deleted_at ON review_like(deleted_at);

-- Table: review_dislike
CREATE TABLE review_dislike (
    id_dislike_review BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_review BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_review_dislike_pengguna FOREIGN KEY (id_pengguna) REFERENCES pengguna(id_user),
    CONSTRAINT fk_review_dislike_review FOREIGN KEY (id_review) REFERENCES review(id_review)
);

CREATE INDEX idx_review_dislike_deleted_at ON review_dislike(deleted_at);

-- ==================================================================================
-- ENGAGEMENT SELLER
-- ==================================================================================

-- Table: jenis_seller_validation
CREATE TABLE jenis_seller_validation (
    id_jenis_seller BIGSERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    validation_status VARCHAR(20) NOT NULL DEFAULT 'Pending',
    alasan_seller TEXT,
    alasan_admin TEXT,
    target_jenis jenis_seller,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_jenis_seller_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller)
);

CREATE INDEX idx_jenis_seller_validation_deleted_at ON jenis_seller_validation(deleted_at);

-- Table: batal_transaksi
CREATE TABLE batal_transaksi (
    id_batal_transaksi BIGSERIAL PRIMARY KEY,
    id_transaksi BIGINT NOT NULL,
    dibatalkan_oleh VARCHAR(20) NOT NULL,
    alasan TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_batal_transaksi_deleted_at ON batal_transaksi(deleted_at);

-- Table: follower
CREATE TABLE follower (
    id_follow BIGSERIAL PRIMARY KEY,
    id_follower BIGINT NOT NULL,
    id_followed BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_follower_pengguna FOREIGN KEY (id_follower) REFERENCES pengguna(id_user),
    CONSTRAINT fk_follower_seller FOREIGN KEY (id_followed) REFERENCES seller(id_seller)
);

CREATE INDEX idx_follower_deleted_at ON follower(deleted_at);
CREATE INDEX idx_follower_user ON follower(id_follower);
CREATE INDEX idx_follower_seller ON follower(id_followed);

-- Table: rekening_seller
CREATE TABLE rekening_seller (
    id_rekening_seller BIGSERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    nama_bank VARCHAR(50) NOT NULL,
    nomor_rekening VARCHAR(50) NOT NULL,
    pemilik_rekening VARCHAR(100) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_rekening_seller_deleted_at ON rekening_seller(deleted_at);
CREATE INDEX idx_rekening_seller_seller ON rekening_seller(id_seller);

-- Table: alamat_gudang
CREATE TABLE alamat_gudang (
    id_alamat_gudang BIGSERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    panggilan_alamat VARCHAR(250) NOT NULL,
    nomor_telefon VARCHAR(20) NOT NULL,
    nama_alamat TEXT NOT NULL,
    provinsi VARCHAR(100) NOT NULL,
    kota VARCHAR(100) NOT NULL,
    kode_pos VARCHAR(40) NOT NULL,
    kode_negara VARCHAR(10) NOT NULL DEFAULT 'IDN',
    deskripsi TEXT,
    longitude DECIMAL(11,8),
    latitude DECIMAL(11,8),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_alamat_gudang_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller)
);

CREATE INDEX idx_alamat_gudang_deleted_at ON alamat_gudang(deleted_at);
CREATE INDEX idx_alamat_gudang_seller ON alamat_gudang(id_seller);

-- Table: distributor_data
CREATE TABLE distributor_data (
    id_distributor_data BIGSERIAL PRIMARY KEY,
    seller_id INT NOT NULL,
    nama_perusahaan TEXT NOT NULL,
    nib VARCHAR(20) NOT NULL,
    npwp VARCHAR(30) NOT NULL,
    dokumen_izin_distributor TEXT NOT NULL,
    alasan TEXT,
    status status_jenis_seller NOT NULL,
    CONSTRAINT fk_distributor_seller FOREIGN KEY (seller_id) REFERENCES seller(id_seller)
);

-- Table: brand_data
CREATE TABLE brand_data (
    id_brand_data BIGSERIAL PRIMARY KEY,
    seller_id INT NOT NULL,
    nama_perusahaan TEXT NOT NULL,
    negara_asal VARCHAR(57) NOT NULL DEFAULT 'indonesia',
    lembaga_pendaftaran TEXT NOT NULL,
    nomor_pendaftaran_merek TEXT NOT NULL,
    sertifikat_merek_url TEXT NOT NULL,
    dokumen_perwakilan_url TEXT NOT NULL,
    nib VARCHAR(20) NOT NULL,
    npwp VARCHAR(30) NOT NULL,
    alasan TEXT,
    status status_jenis_seller NOT NULL,
    CONSTRAINT fk_brand_seller FOREIGN KEY (seller_id) REFERENCES seller(id_seller)
);

-- Table: etalase
CREATE TABLE etalase (
    id_etalase BIGSERIAL PRIMARY KEY,
    id_seller BIGINT NOT NULL,
    nama VARCHAR(100) NOT NULL,
    deskripsi TEXT,
    jumlah_barang INT NOT NULL DEFAULT 0
);

-- Table: barang_ke_etalase
CREATE TABLE barang_ke_etalase (
    id_barang_ke_etalase BIGSERIAL PRIMARY KEY,
    id_etalase BIGINT NOT NULL,
    id_barang_induk BIGINT NOT NULL,
    CONSTRAINT fk_barang_ke_etalase_etalase FOREIGN KEY (id_etalase) 
        REFERENCES etalase(id_etalase) ON UPDATE CASCADE ON DELETE CASCADE
);

-- Table: diskon_produk
CREATE TABLE diskon_produk (
    id_diskon_produk BIGSERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    nama VARCHAR(100) NOT NULL,
    deskripsi TEXT,
    diskon_persen DECIMAL(5,2) NOT NULL,
    berlaku_mulai DATE NOT NULL,
    berlaku_sampai DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Draft',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_diskon_produk_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller)
);

CREATE INDEX idx_diskon_produk_deleted_at ON diskon_produk(deleted_at);

-- Table: barang_di_diskon
CREATE TABLE barang_di_diskon (
    id_barang_di_diskon BIGSERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    id_diskon BIGINT NOT NULL,
    id_barang_induk INT NOT NULL,
    id_kategori_barang BIGINT NOT NULL,
    status status_barang_di_diskon NOT NULL DEFAULT 'Waiting',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_barang_di_diskon_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller),
    CONSTRAINT fk_barang_di_diskon_diskon FOREIGN KEY (id_diskon) 
        REFERENCES diskon_produk(id_diskon_produk) ON DELETE CASCADE
);

CREATE INDEX idx_barang_di_diskon_deleted_at ON barang_di_diskon(deleted_at);

-- ==================================================================================
-- ENGAGEMENT KURIR (COURIER ENGAGEMENT)
-- ==================================================================================

-- Table: informasi_kurir
CREATE TABLE informasi_kurir (
    id_informasi_kurir BIGSERIAL PRIMARY KEY,
    id_kurir BIGINT NOT NULL,
    tanggal_lahir TEXT,
    alasan TEXT,
    informasi_ktp BOOLEAN NOT NULL DEFAULT FALSE,
    informasi_sim BOOLEAN NOT NULL DEFAULT FALSE,
    status status_perizinan NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_informasi_kurir FOREIGN KEY (id_kurir) REFERENCES kurir(id_kurir)
);

CREATE INDEX idx_informasi_kurir_deleted_at ON informasi_kurir(deleted_at);

-- Table: informasi_kendaraan_kurir
CREATE TABLE informasi_kendaraan_kurir (
    id_informasi_kendaraan_kurir BIGSERIAL PRIMARY KEY,
    id_kurir BIGINT NOT NULL,
    jenis_kendaraan jenis_kendaraan_kurir NOT NULL DEFAULT 'Motor',
    nama_kendaraan TEXT NOT NULL,
    roda_kendaraan roda_kendaraan_kurir NOT NULL,
    informasi_stnk BOOLEAN NOT NULL DEFAULT FALSE,
    informasi_bpkb BOOLEAN NOT NULL DEFAULT FALSE,
    nomor_rangka TEXT,
    nomor_mesin TEXT,
    status status_perizinan NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_informasi_kendaraan_kurir FOREIGN KEY (id_kurir) REFERENCES kurir(id_kurir)
);

CREATE INDEX idx_informasi_kendaraan_kurir_deleted_at ON informasi_kendaraan_kurir(deleted_at);

-- Table: alamat_kurir
CREATE TABLE alamat_kurir (
    id_alamat_kurir BIGSERIAL PRIMARY KEY,
    id_kurir BIGINT NOT NULL,
    panggilan_alamat TEXT NOT NULL,
    nomor_telefon VARCHAR(25) NOT NULL,
    nama_alamat TEXT,
    provinsi VARCHAR(100) NOT NULL,
    kota VARCHAR(100) NOT NULL,
    kode_negara VARCHAR(10),
    kode_pos VARCHAR(10),
    deskripsi TEXT,
    longitude DECIMAL(11,8),
    latitude DECIMAL(11,8),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_alamat_kurir FOREIGN KEY (id_kurir) REFERENCES kurir(id_kurir)
);

CREATE INDEX idx_alamat_kurir_deleted_at ON alamat_kurir(deleted_at);

-- Table: rekening_kurir
CREATE TABLE rekening_kurir (
    id_rekening_kurir BIGSERIAL PRIMARY KEY,
    id_kurir BIGINT NOT NULL,
    nama_bank VARCHAR(50) NOT NULL,
    nomor_rekening VARCHAR(50) NOT NULL,
    pemilik_rekening VARCHAR(30) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_rekening_kurir FOREIGN KEY (id_kurir) REFERENCES kurir(id_kurir)
);

CREATE INDEX idx_rekening_kurir_deleted_at ON rekening_kurir(deleted_at);

-- Table: saldo_kurir
CREATE TABLE saldo_kurir (
    id_saldo_kurir BIGSERIAL PRIMARY KEY,
    id_kurir BIGINT NOT NULL,
    saldo BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_saldo_kurir FOREIGN KEY (id_kurir) REFERENCES kurir(id_kurir)
);

CREATE INDEX idx_saldo_kurir_deleted_at ON saldo_kurir(deleted_at);
CREATE UNIQUE INDEX idx_saldo_kurir_unique ON saldo_kurir(id_kurir) WHERE deleted_at IS NULL;

-- Table: bid_kurir_data
CREATE TABLE bid_kurir_data (
    id_bid_kurir_data BIGSERIAL PRIMARY KEY,
    id_kurir BIGINT NOT NULL,
    mode mode_bid_kurir NOT NULL,
    alamat TEXT,
    longitude NUMERIC(11,8) NOT NULL,
    latitude NUMERIC(11,8) NOT NULL,
    max_jarak SMALLINT NOT NULL,
    max_radius SMALLINT,
    max_kg SMALLINT NOT NULL,
    booked_pengiriman INT NOT NULL DEFAULT 0,
    dimulai TIME NOT NULL,
    selesai TIME,
    jenis_kendaraan jenis_kendaraan_kurir NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_bid_kurir_kurir FOREIGN KEY (id_kurir) REFERENCES kurir(id_kurir)
);

CREATE INDEX idx_bid_kurir_data_deleted_at ON bid_kurir_data(deleted_at);
CREATE INDEX idx_bid_kurir_data_kurir ON bid_kurir_data(id_kurir);

-- Apply triggers to engagement tables
CREATE TRIGGER update_entity_social_media_updated_at BEFORE UPDATE ON entity_social_media
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_komentar_updated_at BEFORE UPDATE ON komentar
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_komentar_child_updated_at BEFORE UPDATE ON komentar_child
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_keranjang_updated_at BEFORE UPDATE ON keranjang
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_barang_disukai_updated_at BEFORE UPDATE ON barang_disukai
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wishlist_updated_at BEFORE UPDATE ON wishlist
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alamat_pengguna_updated_at BEFORE UPDATE ON alamat_pengguna
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_review_updated_at BEFORE UPDATE ON review
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_review_like_updated_at BEFORE UPDATE ON review_like
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_review_dislike_updated_at BEFORE UPDATE ON review_dislike
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jenis_seller_validation_updated_at BEFORE UPDATE ON jenis_seller_validation
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_batal_transaksi_updated_at BEFORE UPDATE ON batal_transaksi
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_follower_updated_at BEFORE UPDATE ON follower
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rekening_seller_updated_at BEFORE UPDATE ON rekening_seller
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alamat_gudang_updated_at BEFORE UPDATE ON alamat_gudang
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_diskon_produk_updated_at BEFORE UPDATE ON diskon_produk
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_barang_di_diskon_updated_at BEFORE UPDATE ON barang_di_diskon
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_informasi_kurir_updated_at BEFORE UPDATE ON informasi_kurir
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_informasi_kendaraan_kurir_updated_at BEFORE UPDATE ON informasi_kendaraan_kurir
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alamat_kurir_updated_at BEFORE UPDATE ON alamat_kurir
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rekening_kurir_updated_at BEFORE UPDATE ON rekening_kurir
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_saldo_kurir_updated_at BEFORE UPDATE ON saldo_kurir
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bid_kurir_data_updated_at BEFORE UPDATE ON bid_kurir_data
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================================================================================
-- PRODUCT TABLES (BARANG)
-- ==================================================================================

-- Additional ENUM types for product tables
CREATE TYPE status_varian AS ENUM ('Ready', 'Dipesan', 'Diproses', 'Terjual');

-- Table: barang_induk (Parent Product)
CREATE TABLE barang_induk (
    id_barang_induk SERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    id_diskon BIGINT,
    nama_barang VARCHAR(200) NOT NULL,
    jenis_barang seller_dedication NOT NULL DEFAULT 'Semua Barang',
    deskripsi TEXT,
    original_kategori BIGINT,
    viewed INT NOT NULL DEFAULT 0,
    likes INT NOT NULL DEFAULT 0,
    total_komentar INT NOT NULL DEFAULT 0,
    harga_kategori_barang INT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_barang_induk_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller)
);

CREATE INDEX idx_barang_induk_deleted_at ON barang_induk(deleted_at);
CREATE INDEX idx_barang_induk_seller ON barang_induk(id_seller);
CREATE INDEX idx_barang_induk_jenis ON barang_induk(jenis_barang);

-- Table: kategori_barang (Product Category/Variant)
CREATE TABLE kategori_barang (
    id_kategori_barang BIGSERIAL PRIMARY KEY,
    id_seller INT NOT NULL,
    id_barang_induk INT NOT NULL,
    id_alamat_gudang BIGINT,
    id_rekening BIGINT,
    nama VARCHAR(120) NOT NULL,
    deskripsi TEXT,
    warna VARCHAR(50),
    stok INT NOT NULL,
    harga INT NOT NULL,
    diskon INT DEFAULT 0,
    berat_gram SMALLINT,
    dimensi_panjang_cm SMALLINT,
    dimensi_lebar_cm SMALLINT,
    sku VARCHAR(20) NOT NULL,
    is_original BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_kategori_barang_seller FOREIGN KEY (id_seller) REFERENCES seller(id_seller),
    CONSTRAINT fk_kategori_barang_induk FOREIGN KEY (id_barang_induk) 
        REFERENCES barang_induk(id_barang_induk) ON DELETE CASCADE,
    CONSTRAINT fk_kategori_barang_alamat FOREIGN KEY (id_alamat_gudang) 
        REFERENCES alamat_gudang(id_alamat_gudang),
    CONSTRAINT fk_kategori_barang_rekening FOREIGN KEY (id_rekening) 
        REFERENCES rekening_seller(id_rekening_seller)
);

CREATE INDEX idx_kategori_barang_deleted_at ON kategori_barang(deleted_at);
CREATE INDEX idx_kategori_barang_induk ON kategori_barang(id_barang_induk);
CREATE INDEX idx_kategori_barang_seller ON kategori_barang(id_seller);
CREATE INDEX idx_kategori_barang_sku ON kategori_barang(sku);

-- Table: varian_barang (Product Variant/Individual Item)
CREATE TABLE varian_barang (
    id_varian_barang BIGSERIAL PRIMARY KEY,
    id_barang_induk INT NOT NULL,
    id_kategori BIGINT NOT NULL,
    id_transaksi BIGINT,
    sku VARCHAR(100) NOT NULL,
    status status_varian NOT NULL DEFAULT 'Ready',
    hold_by BIGINT DEFAULT 0,
    holder_entity VARCHAR(30),
    CONSTRAINT fk_varian_barang_induk FOREIGN KEY (id_barang_induk) 
        REFERENCES barang_induk(id_barang_induk),
    CONSTRAINT fk_varian_barang_kategori FOREIGN KEY (id_kategori) 
        REFERENCES kategori_barang(id_kategori_barang) ON DELETE CASCADE
);

CREATE INDEX idx_varian_barang_induk ON varian_barang(id_barang_induk);
CREATE INDEX idx_varian_barang_kategori ON varian_barang(id_kategori);
CREATE INDEX idx_varian_barang_status ON varian_barang(status);
CREATE INDEX idx_varian_barang_sku ON varian_barang(sku);
CREATE INDEX idx_varian_barang_transaksi ON varian_barang(id_transaksi);

-- Apply auto-update triggers
CREATE TRIGGER update_barang_induk_updated_at BEFORE UPDATE ON barang_induk
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_kategori_barang_updated_at BEFORE UPDATE ON kategori_barang
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add foreign key constraints from engagement tables that reference barang_induk
-- (These assume barang_induk already exists from previous migrations)
ALTER TABLE komentar 
    ADD CONSTRAINT fk_komentar_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE keranjang 
    ADD CONSTRAINT fk_keranjang_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE keranjang 
    ADD CONSTRAINT fk_keranjang_kategori 
    FOREIGN KEY (id_kategori_barang) REFERENCES kategori_barang(id_kategori_barang);

ALTER TABLE barang_disukai 
    ADD CONSTRAINT fk_barang_disukai_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE wishlist 
    ADD CONSTRAINT fk_wishlist_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE review 
    ADD CONSTRAINT fk_review_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE barang_ke_etalase 
    ADD CONSTRAINT fk_barang_ke_etalase_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE barang_di_diskon 
    ADD CONSTRAINT fk_barang_di_diskon_barang_induk 
    FOREIGN KEY (id_barang_induk) REFERENCES barang_induk(id_barang_induk);

ALTER TABLE barang_di_diskon 
    ADD CONSTRAINT fk_barang_di_diskon_kategori 
    FOREIGN KEY (id_kategori_barang) REFERENCES kategori_barang(id_kategori_barang);

-- Add foreign key from barang_induk to diskon_produk
ALTER TABLE barang_induk 
    ADD CONSTRAINT fk_barang_induk_diskon 
    FOREIGN KEY (id_diskon) REFERENCES diskon_produk(id_diskon_produk);

-- Comments for documentation
COMMENT ON TABLE barang_induk IS 'Parent product table containing main product information';
COMMENT ON TABLE kategori_barang IS 'Product categories/variants with pricing, stock, and specifications';
COMMENT ON TABLE varian_barang IS 'Individual product items tracking specific inventory units';

COMMENT ON COLUMN kategori_barang.is_original IS 'Flag indicating if this is the original/default category for the product';
COMMENT ON COLUMN varian_barang.hold_by IS 'ID of entity (user/transaction) holding this item';
COMMENT ON COLUMN varian_barang.holder_entity IS 'Type of entity holding the item (e.g., user, transaction)';
COMMENT ON COLUMN barang_induk.harga_kategori_barang IS 'Base price from primary category';

-- ==================================================================================
-- TRANSACTION & PAYMENT TABLES
-- ==================================================================================

-- Additional ENUM types for transaction tables
CREATE TYPE status_transaksi AS ENUM (
    'Dibayar', 
    'Diproses', 
    'Dikirim', 
    'Selesai', 
    'Dibatalkan',
    'Dikembalikan'
);

CREATE TYPE jenis_entity AS ENUM ('Pengguna', 'Seller', 'Kurir', 'Admin', 'System');

CREATE TYPE status_paid_failed AS ENUM ('Pending', 'Expired', 'Cancelled', 'Failed');

-- Table: pembayaran (Payment)
CREATE TABLE pembayaran (
    id_pembayaran BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    kode_transaksi_pg VARCHAR(250) NOT NULL,
    kode_order_sistem VARCHAR(250) UNIQUE NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    total INT NOT NULL DEFAULT 0,
    payment_type VARCHAR(120) NOT NULL,
    paid_at TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_pembayaran_pengguna FOREIGN KEY (id_pengguna) 
        REFERENCES pengguna(id_user)
);

CREATE INDEX idx_pembayaran_deleted_at ON pembayaran(deleted_at);
CREATE INDEX idx_pembayaran_pengguna ON pembayaran(id_pengguna);
CREATE INDEX idx_pembayaran_kode_order ON pembayaran(kode_order_sistem);
CREATE INDEX idx_pembayaran_kode_pg ON pembayaran(kode_transaksi_pg);

-- Table: transaksi (Transaction)
CREATE TABLE transaksi (
    id_transaksi BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    id_seller INT NOT NULL,
    id_barang_induk BIGINT NOT NULL,
    id_kategori_barang BIGINT NOT NULL,
    id_alamat_pengguna BIGINT NOT NULL,
    id_alamat_gudang BIGINT NOT NULL,
    id_alamat_ekspedisi BIGINT NOT NULL,
    id_pembayaran BIGINT NOT NULL,
    kendaraan_pengiriman jenis_kendaraan_kurir DEFAULT 'Motor' NOT NULL,
    jenis_pengiriman jenis_layanan_kurir NOT NULL,
    jarak_tempuh VARCHAR(100) NOT NULL,
    berat_total_kg SMALLINT NOT NULL,
    kode_order_sistem VARCHAR(100) NOT NULL,
    kode_resi_ekspedisi VARCHAR(100),
    status status_transaksi DEFAULT 'Dibayar' NOT NULL,
    dibatalkan_oleh jenis_entity,
    catatan TEXT,
    kuantitas_barang INT NOT NULL,
    is_ekspedisi BOOLEAN NOT NULL DEFAULT FALSE,
    seller_paid BIGINT NOT NULL,
    kurir_paid BIGINT NOT NULL,
    ekspedisi_paid BIGINT NOT NULL,
    total BIGINT NOT NULL,
    reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_transaksi_pengguna FOREIGN KEY (id_pengguna) 
        REFERENCES pengguna(id_user),
    CONSTRAINT fk_transaksi_seller FOREIGN KEY (id_seller) 
        REFERENCES seller(id_seller),
    CONSTRAINT fk_transaksi_barang_induk FOREIGN KEY (id_barang_induk) 
        REFERENCES barang_induk(id_barang_induk),
    CONSTRAINT fk_transaksi_kategori_barang FOREIGN KEY (id_kategori_barang) 
        REFERENCES kategori_barang(id_kategori_barang),
    CONSTRAINT fk_transaksi_alamat_pengguna FOREIGN KEY (id_alamat_pengguna) 
        REFERENCES alamat_pengguna(id_alamat_user),
    CONSTRAINT fk_transaksi_alamat_gudang FOREIGN KEY (id_alamat_gudang) 
        REFERENCES alamat_gudang(id_alamat_gudang),
    CONSTRAINT fk_transaksi_pembayaran FOREIGN KEY (id_pembayaran) 
        REFERENCES pembayaran(id_pembayaran)
);

CREATE INDEX idx_transaksi_deleted_at ON transaksi(deleted_at);
CREATE INDEX idx_transaksi_pengguna ON transaksi(id_pengguna);
CREATE INDEX idx_transaksi_seller ON transaksi(id_seller);
CREATE INDEX idx_transaksi_barang ON transaksi(id_barang_induk);
CREATE INDEX idx_transaksi_status ON transaksi(status);
CREATE INDEX idx_transaksi_kode_order ON transaksi(kode_order_sistem);
CREATE INDEX idx_transaksi_kode_resi ON transaksi(kode_resi_ekspedisi);
CREATE INDEX idx_transaksi_pembayaran ON transaksi(id_pembayaran);

-- Table: pembayaran_failed (Failed Payment)
CREATE TABLE pembayaran_failed (
    id_paid_failed BIGSERIAL PRIMARY KEY,
    id_pengguna BIGINT NOT NULL,
    finish_redirect_url TEXT,
    fraud_status VARCHAR(50),
    gross_amount VARCHAR(50),
    order_id VARCHAR(100),
    payment_type VARCHAR(50),
    pdf_url TEXT,
    status_code VARCHAR(10),
    status_message TEXT,
    transaction_id VARCHAR(100),
    transaction_status VARCHAR(50),
    transaction_time VARCHAR(50),
    bank VARCHAR(50),
    va_number VARCHAR(50),
    payment_code VARCHAR(50),
    status status_paid_failed DEFAULT 'Pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_pembayaran_failed_pengguna FOREIGN KEY (id_pengguna) 
        REFERENCES pengguna(id_user)
);

CREATE INDEX idx_pembayaran_failed_deleted_at ON pembayaran_failed(deleted_at);
CREATE INDEX idx_pembayaran_failed_pengguna ON pembayaran_failed(id_pengguna);
CREATE INDEX idx_pembayaran_failed_order_id ON pembayaran_failed(order_id);
CREATE INDEX idx_pembayaran_failed_transaction_id ON pembayaran_failed(transaction_id);
CREATE INDEX idx_pembayaran_failed_status ON pembayaran_failed(status);

-- Table: transaksi_failed (Failed Transaction)
CREATE TABLE transaksi_failed (
    id_transaksi_failed BIGSERIAL PRIMARY KEY,
    id_pembayaran_failed BIGINT NOT NULL,
    id_pengguna BIGINT NOT NULL,
    id_seller INT NOT NULL,
    id_barang_induk INT NOT NULL,
    id_kategori_barang BIGINT NOT NULL,
    id_alamat_pengguna BIGINT,
    jenis_pengiriman VARCHAR(50) NOT NULL DEFAULT 'reguler',
    catatan TEXT,
    kuantitas_barang SMALLINT NOT NULL,
    total BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_transaksi_failed_pembayaran FOREIGN KEY (id_pembayaran_failed) 
        REFERENCES pembayaran_failed(id_paid_failed),
    CONSTRAINT fk_transaksi_failed_pengguna FOREIGN KEY (id_pengguna) 
        REFERENCES pengguna(id_user),
    CONSTRAINT fk_transaksi_failed_seller FOREIGN KEY (id_seller) 
        REFERENCES seller(id_seller),
    CONSTRAINT fk_transaksi_failed_barang_induk FOREIGN KEY (id_barang_induk) 
        REFERENCES barang_induk(id_barang_induk),
    CONSTRAINT fk_transaksi_failed_kategori_barang FOREIGN KEY (id_kategori_barang) 
        REFERENCES kategori_barang(id_kategori_barang)
);

CREATE INDEX idx_transaksi_failed_deleted_at ON transaksi_failed(deleted_at);
CREATE INDEX idx_transaksi_failed_pembayaran ON transaksi_failed(id_pembayaran_failed);
CREATE INDEX idx_transaksi_failed_pengguna ON transaksi_failed(id_pengguna);
CREATE INDEX idx_transaksi_failed_seller ON transaksi_failed(id_seller);

-- Add foreign key from batal_transaksi to transaksi
ALTER TABLE batal_transaksi 
    ADD CONSTRAINT fk_batal_transaksi_transaksi 
    FOREIGN KEY (id_transaksi) REFERENCES transaksi(id_transaksi);

-- Add foreign key from varian_barang to transaksi
ALTER TABLE varian_barang 
    ADD CONSTRAINT fk_varian_barang_transaksi 
    FOREIGN KEY (id_transaksi) REFERENCES transaksi(id_transaksi);

-- Apply auto-update triggers
CREATE TRIGGER update_pembayaran_updated_at BEFORE UPDATE ON pembayaran
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transaksi_updated_at BEFORE UPDATE ON transaksi
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pembayaran_failed_updated_at BEFORE UPDATE ON pembayaran_failed
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transaksi_failed_updated_at BEFORE UPDATE ON transaksi_failed
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE pembayaran IS 'Payment records from payment gateway';
COMMENT ON TABLE transaksi IS 'Main transaction records for successful orders';
COMMENT ON TABLE pembayaran_failed IS 'Failed payment attempts with payment gateway details';
COMMENT ON TABLE transaksi_failed IS 'Failed transaction records linked to failed payments';

COMMENT ON COLUMN transaksi.seller_paid IS 'Amount paid to seller';
COMMENT ON COLUMN transaksi.kurir_paid IS 'Amount paid to courier';
COMMENT ON COLUMN transaksi.ekspedisi_paid IS 'Amount paid to expedition service';
COMMENT ON COLUMN transaksi.is_ekspedisi IS 'Flag indicating if using expedition service';
COMMENT ON COLUMN transaksi.reviewed IS 'Flag indicating if transaction has been reviewed by user';
COMMENT ON COLUMN transaksi.kode_order_sistem IS 'Internal system order code';
COMMENT ON COLUMN transaksi.kode_resi_ekspedisi IS 'Tracking number from courier/expedition';
COMMENT ON COLUMN pembayaran.kode_transaksi_pg IS 'Transaction code from payment gateway';
COMMENT ON COLUMN pembayaran.kode_order_sistem IS 'Internal system order code matching transaction';
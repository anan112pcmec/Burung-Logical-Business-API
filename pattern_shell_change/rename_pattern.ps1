Set-Location -Path 'c:\Burung_App\Project_Source\Backend-1'

$files = @(
    # 'app\service\kurir_services\alamat_services\engagement_alamat.go',
    # 'app\service\kurir_services\credential_services\engagement_password.go',
    # 'app\service\kurir_services\informasi_services\engagement_informasi.go',
    # 'app\service\kurir_services\media_services\engagement_media.go',
    # 'app\service\kurir_services\pengiriman_services\engagement_pengiriman.go'
    # 'app\service\kurir_services\profiling_services\particular_profiling\function.go',
    # 'app\service\kurir_services\profiling_services\engagament_profiling.go',
    # 'app\service\kurir_services\rekening_services\engagement_rekening.go',
    # 'app\service\kurir_services\social_media_services\engagement_social_media.go'

    # 'app\service\pengguna_service\alamat_services\engagement_alamat.go',
    # 'app\service\pengguna_service\barang_services\engagement_barang.go',
    # 'app\service\pengguna_service\credential_services\engagement_password.go',
    # 'app\service\pengguna_service\media_services\engagement_media.go',
    # 'app\service\pengguna_service\profiling_services\particular_profiling\function.go',
    # 'app\service\pengguna_service\profiling_services\engagement_profile.go',
    # 'app\service\pengguna_service\social_media_services\engagement_social_media.go',
    # 'app\service\pengguna_service\transaction_services\engagement_transaksi.go',
    # 'app\service\pengguna_service\wishlist_services\engagement_wishlist.go'

    'app\service\seller_services\alamat_services\engagement_alamat.go',
    'app\service\seller_services\barang_services\engagement_barang.go',
    'app\service\seller_services\credential_services\engagament_password.go',
    'app\service\seller_services\credential_services\engagement_rekening.go',
    'app\service\seller_services\diskon_services\engagement_diskon.go',
    'app\service\seller_services\etalase_services\engagement_etalase.go',
    'app\service\seller_services\jenis_seller_services\engagement_jenis.go',
    'app\service\seller_services\media_services\engagement_media.go',
    'app\service\seller_services\profiling_services\particular_profiling\function.go',
    'app\service\seller_services\profiling_services\engagement_profiling.go',
    'app\service\seller_services\social_media_services\engagement_social_media.go',
    'app\service\seller_services\transaksi_services\engagement_transaksi.go'
)

foreach ($file in $files) {
    if (Test-Path -Path $file) {
        $content = Get-Content -Raw -Path $file
        
        # Regex untuk mendeteksi '5 * time.Second' atau 'time.Second * 5' (fleksibel spasi)
        # \s* artinya spasi boleh ada atau tidak (misal: 5*time atau 5 * time)
        $pattern = '(([1-9]|10)\s*\*\s*time\.Second|time\.Second\s*\*\s*([1-9]|10))'
        
        # Lakukan replace menggunakan regex pattern di atas
        $content = $content -replace $pattern, 'settings.TimeoutContext'
        
        # Di Go, biasakan pakai UTF8 tanpa BOM agar compiler tidak protes
        [System.IO.File]::WriteAllText((Resolve-Path $file), $content, (New-Object System.Text.UTF8Encoding($false)))
        
        Write-Host "Updated $file" -ForegroundColor Green
    } else {
        Write-Warning "File tidak ditemukan: $file"
    }
}

Write-Host "Proses selesai!" -ForegroundColor Cyan
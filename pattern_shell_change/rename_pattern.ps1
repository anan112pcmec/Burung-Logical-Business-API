Set-Location -Path 'c:\Burung_App\Project_Source\Backend-1'

$files = @(
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\alamat_services\engagement_alamat.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\credential_services\engagement_password.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\informasi_services\engagement_informasi.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\media_services\engagement_media.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\pengiriman_services\engagement_pengiriman.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\profiling_services\particular_profiling\function.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\profiling_services\engagament_profiling.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\rekening_services\engagement_rekening.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\social_media_services\engagement_social_media.go',

    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\alamat_services\engagement_alamat.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\barang_services\engagement_barang.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\credential_services\engagement_password.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\media_services\engagement_media.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\profiling_services\particular_profiling\function.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\profiling_services\engagement_profile.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\social_media_services\engagement_social_media.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\transaction_services\engagement_transaksi.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\wishlist_services\engagement_wishlist.go',

    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\alamat_services\engagement_alamat.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\barang_services\engagement_barang.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\credential_services\engagament_password.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\credential_services\engagement_rekening.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\diskon_services\engagement_diskon.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\etalase_services\engagement_etalase.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\jenis_seller_services\engagement_jenis.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\media_services\engagement_media.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\profiling_services\particular_profiling\function.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\profiling_services\engagement_profiling.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\social_media_services\engagement_social_media.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\transaksi_services\engagement_transaksi.go',

    # 'C:\Burung_App\Project_Source\Backend-1\app\database\sot_database\threshold\engagement_entity_threshold.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\database\cache_database\entity_sessioning\seeders\key.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\initialize.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\database\cache_database\entity_sessioning\seeders\update.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\database\sot_database\migrate\sot_up_migrate.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\helper\helper.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\callback\payment_out\callback_function.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\cache\data\pengiriman.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\cache\maintain\maintain.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\api\payment_in_midtrans\gerai\gerai_contract.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\api\payment_in_midtrans\virtual_account\virtual_acc_contract.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\api\payment_in_midtrans\wallet\wallet_contract.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\routes\auth\auth.go',
    # 'C:\Burung_App\Project_Source\Backend-1\app\initialize.go'

    'C:\Burung_App\Project_Source\Backend-1\app\service\seller_services\identity_seller\type.go',
    'C:\Burung_App\Project_Source\Backend-1\app\service\pengguna_service\identity_pengguna\type.go',
    'C:\Burung_App\Project_Source\Backend-1\app\service\kurir_services\identity_kurir\type.go',
    'C:\Burung_App\Project_Source\Backend-1\app\service\authservices\auth.go'
)

foreach ($file in $files) {
    if (Test-Path -Path $file) {
        $content = Get-Content -Raw -Path $file
        
        # Regex pola durasi angka 1-10 dikali dengan time.Second
        # $pattern = '(([1-9]|10)\s*\*\s*time\.Second|time\.Second\s*\*\s*([1-9]|10))'
        
        $content = $content -replace 'models', 'sot_models'
        
        # PERBAIKAN: Langsung lempar variable $file ke WriteAllText tanpa dibungkus Resolve-Path
        [System.IO.File]::WriteAllText($file, $content, (New-Object System.Text.UTF8Encoding($false)))
        
        Write-Host "Updated $file" -ForegroundColor Green
    } else {
        Write-Warning "File tidak ditemukan: $file"
    }
}

Write-Host "Proses selesai!" -ForegroundColor Cyan
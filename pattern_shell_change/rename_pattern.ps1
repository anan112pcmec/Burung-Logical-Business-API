Set-Location -Path 'c:\Burung_App\Project_Source\Backend-1'

$files = @(
    'app\service\seller_services\credential_services\engagement_rekening.go'

    
    
)

foreach ($file in $files) {
    if (Test-Path -Path $file) {
        $content = Get-Content -Raw -Path $file
        
        # Mengganti semua kata ParseToInsertType menjadi ParseToCUDType
        $content = $content -replace 'config.InternalDBReadWriteSystem', 'environment.InternalDBReadWriteSystem'
        
        Set-Content -Path $file -Value $content -Encoding utf8
        Write-Host "Updated $file"
    } else {
        Write-Warning "File tidak ditemukan: $file"
    }
}

Write-Host "Proses selesai!"
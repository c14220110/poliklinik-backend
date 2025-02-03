-- 1. Tabel tanpa Foreign Key
CREATE TABLE `Administrasi` (
  `ID_Admin` int(11) NOT NULL AUTO_INCREMENT,
  `Nama` varchar(255) NOT NULL,
  `Username` varchar(100) NOT NULL,
  `Password` varchar(255) NOT NULL,
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Admin`),
  UNIQUE KEY `Username` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `Dokter` (
  `ID_Dokter` int(11) NOT NULL AUTO_INCREMENT,
  `Nama` varchar(255) NOT NULL,
  `Username` varchar(100) NOT NULL,
  `Password` varchar(255) NOT NULL,
  `Spesialisasi` varchar(255) NOT NULL,
  PRIMARY KEY (`ID_Dokter`),
  UNIQUE KEY `Username` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `Pasien` (
  `ID_Pasien` int(11) NOT NULL AUTO_INCREMENT,
  `Nama` varchar(255) NOT NULL,
  `Tanggal_Lahir` date NOT NULL,
  `Alamat` text DEFAULT NULL,
  `No_Telp` varchar(20) DEFAULT NULL,
  `Poli_Tujuan` varchar(255) NOT NULL,
  `Tanggal_Registrasi` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Pasien`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `Poliklinik` (
  `ID_Poli` int(11) NOT NULL AUTO_INCREMENT,
  `Nama_Poli` varchar(255) NOT NULL,
  `Jumlah_Tenkes` int(11) NOT NULL,
  `Alamat` text DEFAULT NULL,
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Poli`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `Suster` (
  `ID_Suster` int(11) NOT NULL AUTO_INCREMENT,
  `Nama` varchar(255) NOT NULL,
  `Username` varchar(100) NOT NULL,
  `Password` varchar(255) NOT NULL,
  `Poli_Tugas` varchar(255) NOT NULL,
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Suster`),
  UNIQUE KEY `Username` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 2. Tabel dengan FK ke tabel Poliklinik
CREATE TABLE `Jadwal_Shift` (
  `ID_Shift` int(11) NOT NULL AUTO_INCREMENT,
  `ID_Poli` int(11) NOT NULL,
  `Jam_Mulai` datetime NOT NULL,
  `Jam_Selesai` datetime NOT NULL,
  `Assigned_Tenkes` int(11) NOT NULL,
  PRIMARY KEY (`ID_Shift`),
  FOREIGN KEY (`ID_Poli`) REFERENCES `Poliklinik` (`ID_Poli`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 3. Tabel dengan FK ke Pasien dan Poliklinik
CREATE TABLE `Antrian` (
  `ID_Antrian` int(11) NOT NULL AUTO_INCREMENT,
  `ID_Pasien` int(11) NOT NULL,
  `ID_Poli` int(11) NOT NULL,
  `Nomor_Antrian` int(11) NOT NULL,
  `Status` int(11) NOT NULL CHECK (`Status` IN (0,1,2)),
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Antrian`),
  FOREIGN KEY (`ID_Pasien`) REFERENCES `Pasien` (`ID_Pasien`) ON DELETE CASCADE,
  FOREIGN KEY (`ID_Poli`) REFERENCES `Poliklinik` (`ID_Poli`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `Billing` (
  `ID_Billing` int(11) NOT NULL AUTO_INCREMENT,
  `ID_Pasien` int(11) NOT NULL,
  `ID_Admin` int(11) NOT NULL,
  `Status` int(11) NOT NULL CHECK (`Status` IN (0,1)),
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Billing`),
  FOREIGN KEY (`ID_Pasien`) REFERENCES `Pasien` (`ID_Pasien`) ON DELETE CASCADE,
  FOREIGN KEY (`ID_Admin`) REFERENCES `Administrasi` (`ID_Admin`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `E_Resep` (
  `ID_Resep` int(11) NOT NULL AUTO_INCREMENT,
  `ID_Pasien` int(11) NOT NULL,
  `ID_Dokter` int(11) NOT NULL,
  `Nama_Obat` varchar(255) NOT NULL,
  `Jumlah` int(11) NOT NULL,
  `Dosis` varchar(50) NOT NULL,
  `Harga` decimal(10,2) NOT NULL,
  `Keterangan` text DEFAULT NULL,
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Resep`),
  FOREIGN KEY (`ID_Pasien`) REFERENCES `Pasien` (`ID_Pasien`) ON DELETE CASCADE,
  FOREIGN KEY (`ID_Dokter`) REFERENCES `Dokter` (`ID_Dokter`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 4. Tabel dengan FK ke Pasien dan Suster
CREATE TABLE `Screening` (
  `ID_Screening` int(11) NOT NULL AUTO_INCREMENT,
  `ID_Pasien` int(11) NOT NULL,
  `ID_Suster` int(11) NOT NULL,
  `Tensi` int(11) NOT NULL,
  `Berat_Badan` int(11) NOT NULL,
  `Suhu_Tubuh` int(11) NOT NULL,
  `Tinggi_Badan` int(11) NOT NULL,
  `Keterangan` text DEFAULT NULL,
  `Created_At` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`ID_Screening`),
  FOREIGN KEY (`ID_Pasien`) REFERENCES `Pasien` (`ID_Pasien`) ON DELETE CASCADE,
  FOREIGN KEY (`ID_Suster`) REFERENCES `Suster` (`ID_Suster`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 5. Tabel dengan FK ke banyak tabel
CREATE TABLE `Rekam_Medis` (
  `ID_RM` int(11) NOT NULL AUTO_INCREMENT,
  `ID_Pasien` int(11) NOT NULL,
  `ID_Dokter` int(11) NOT NULL,
  `ID_E_Resep` int(11) DEFAULT NULL,
  `ID_Screening` int(11) DEFAULT NULL,
  `ID_Billing` int(11) DEFAULT NULL,
  PRIMARY KEY (`ID_RM`),
  FOREIGN KEY (`ID_Pasien`) REFERENCES `Pasien` (`ID_Pasien`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

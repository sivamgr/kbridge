package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sivamgr/kstreamdb"
)

var db kstreamdb.DB

func setupDatabase() {
	db = kstreamdb.SetupDatabase(AppConfig.DataManagement.DataPath)
	go doPurge()
}

func dirSize(dpath string) int64 {
	size := int64(0)
	filepath.Walk(dpath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size
}

func isDirEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true
	}
	return false
}

func purgeDirBySize(dpath string, size int64) {
	filepath.Walk(dpath, func(fp string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if size > 0 {
				size -= info.Size()
				os.Remove(fp)
			} else {
				return io.EOF
			}
		} else {
			if isDirEmpty(fp) {
				// if directory is empty remove
				os.Remove(fp)
			}
		}
		return err
	})
}

func purgeOldData() {
	db.Compress()
	sizeOverLimit := dirSize(db.DataPath) - int64(AppConfig.DataManagement.StorageLimitInGB)*int64(1024*1024*1024)
	if sizeOverLimit > 0 {
		purgeDirBySize(db.DataPath, sizeOverLimit)
	}
}

func hasGonePast(t string) bool {
	tn := time.Now().Local()
	lt := tn.Format("15:04:05")
	return (lt > t)
}

func doPurge() {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		<-ticker.C
		if hasGonePast(AppConfig.DataManagement.StoragePurgeTime) {
			log.Printf("Data Purge now :%s configured time :%s", AppConfig.DataManagement.StoragePurgeTime, time.Now().Local().Format("15:04:05"))
			purgeOldData()
		}
	}
}

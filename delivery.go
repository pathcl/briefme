package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// DeliverEPUB copies the EPUB to the Kobo mount path.
// If koboPath is empty it attempts auto-detection.
func DeliverEPUB(koboPath, epubPath string) error {
	if koboPath == "" {
		detected, err := detectKoboPath()
		if err != nil {
			return err
		}
		koboPath = detected
	}

	if _, err := os.Stat(koboPath); err != nil {
		return fmt.Errorf("kobo path %q not accessible (is the device plugged in?): %w", koboPath, err)
	}

	dst := filepath.Join(koboPath, filepath.Base(epubPath))
	if err := copyFile(epubPath, dst); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf(
				"permission denied writing to %q\n\n"+
					"The Kobo is mounted without uid= so ownership defaults to root.\n"+
					"Note: 'remount' cannot change uid on FAT — you must unmount and remount:\n\n"+
					"  sudo umount %s\n"+
					"  sudo mount -t vfat -o uid=%d,gid=%d,fmask=0022,dmask=0022 <device> %s\n\n"+
					"For a permanent fix add this to /etc/fstab (replace UUID and device):\n"+
					"  UUID=<uuid>  %s  vfat  uid=%d,gid=%d,fmask=0022,dmask=0022,nofail  0  0\n"+
					"Find the UUID with: sudo blkid <device>",
				koboPath,
				koboPath, os.Getuid(), os.Getgid(), koboPath,
				koboPath, os.Getuid(), os.Getgid(),
			)
		}
		return err
	}
	return nil
}

func detectKoboPath() (string, error) {
	user := os.Getenv("USER")
	candidates := []string{
		"/Volumes/KOBOeReader",                               // macOS
		filepath.Join("/media", user, "KOBOeReader"),         // Linux (udisks)
		filepath.Join("/run/media", user, "KOBOeReader"),     // Linux (systemd)
	}
	if runtime.GOOS == "windows" {
		// On Windows, scan common drive letters for the Kobo marker file.
		for _, drive := range "DEFGHIJKLMNOPQRSTUVWXYZ" {
			p := string(drive) + `:\`
			if _, err := os.Stat(filepath.Join(p, ".kobo")); err == nil {
				return p, nil
			}
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("Kobo not found; connect it via USB or set kobo_path in config.yaml")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

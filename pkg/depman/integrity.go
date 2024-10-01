package depman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/storage"
)

/*
{
    "Version": "v1.0.0",
    "Time": "2023-06-20T06:39:01Z",
    "Origin": {
        "VCS": "git",
        "URL": "https://github.com/imdario/mergo",
        "Hash": "131de815afc35a77c41ae99da6c8f4288b6cb513",
        "Ref": "refs/tags/v1.0.0"
    }
}
*/

type SourceIntegrityInfo struct {
	Version string         `json:"Version"`
	Time    string         `json:"Time"`
	Origin  storage.Origin `json:"Origin"`
}

func (inf *SourceIntegrityInfo) Read(dm *dependencyManager, source string, version string) error {
	infoPath := dm.getSourceInfoPath(source, version)
	if _, err := os.Stat(infoPath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("stat %s: %w", infoPath, err)
	}

	if err := filesys.ReadJSON(infoPath, inf); err != nil {
		return fmt.Errorf("read origin info %s: %w", infoPath, err)
	}

	return nil
}

func (inf *SourceIntegrityInfo) Write(dm *dependencyManager, source string, version string) error {
	infoPath := dm.getSourceInfoPath(source, version)

	if err := os.MkdirAll(filepath.Dir(infoPath), os.ModePerm); err != nil {
		return fmt.Errorf("create bundle info directory: %w", err)
	}

	if err := filesys.WriteJSON(infoPath, inf); err != nil {
		return fmt.Errorf("write %s: %w", infoPath, err)
	}

	return nil
}

type BundleIntegrityInfo struct {
	Source  string `json:"Source"`
	Version string `json:"Version"`
	Hash    string `json:"Hash"`
}

func (inf *BundleIntegrityInfo) Read(dm *dependencyManager, appCode string, version string) error {
	infoPath := dm.getBundleInfoPath(appCode, version)
	if _, err := os.Stat(infoPath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("stat %s: %w", infoPath, err)
	}

	if err := filesys.ReadJSON(infoPath, inf); err != nil {
		return fmt.Errorf("read %s: %w", infoPath, err)
	}

	return nil
}

func (inf *BundleIntegrityInfo) Write(dm *dependencyManager, appCode string, version string) error {
	infoPath := dm.getBundleInfoPath(appCode, version)

	if err := os.MkdirAll(filepath.Dir(infoPath), os.ModePerm); err != nil {
		return fmt.Errorf("create bundle info directory: %w", err)
	}

	if err := filesys.WriteJSON(infoPath, inf); err != nil {
		return fmt.Errorf("write %s: %w", infoPath, err)
	}

	return nil
}

func (dm *dependencyManager) validateSourceInformation(source string, version string, info storage.Origin) error {
	sourceInfo := SourceIntegrityInfo{
		Origin: dm.Storage.Origin(), // required for proper parsing
	}
	if err := sourceInfo.Read(dm, source, version); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read source info: %w", err)
	}

	if err := sourceInfo.Origin.Validate(info); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	return nil
}

// Check source and bundle integrity cache and update both
func (dm *dependencyManager) updateDependencyCache(source string, version string, info storage.Origin, depDir string, depIdx *bundle.Index) error {
	sourceInfo := SourceIntegrityInfo{
		Origin: dm.Storage.Origin(), // required for proper parsing
	}

	if err := sourceInfo.Read(dm, source, version); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read source info: %w", err)
		}

		sourceInfo = SourceIntegrityInfo{
			Version: version,
			Time:    "TODO",
			Origin:  info,
		}

		if err := sourceInfo.Write(dm, source, version); err != nil {
			return fmt.Errorf("write integrity info: %w", err)
		}
	} else {
		// source information already exists
		// TODO validate the information
	}

	// move dependency from cache to the dependencies directory, calculate directory integrity information
	// TODO save additional storage specific information

	bundleInfo := BundleIntegrityInfo{}
	if err := bundleInfo.Read(dm, depIdx.AppCode, version); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read bundle info: %w", err)
		}

		hash, err := filesys.ComputeDirectoryHash(depDir)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		bundleInfo = BundleIntegrityInfo{
			Source:  source,
			Version: version,
			Hash:    hash,
		}

		if err := bundleInfo.Write(dm, depIdx.AppCode, version); err != nil {
			return fmt.Errorf("write bundle integrity info: %w", err)
		}
	} else {
		hash, err := filesys.ComputeDirectoryHash(depDir)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		if hash != bundleInfo.Hash {
			return fmt.Errorf("bundle integrity check failed")
		}
	}

	return nil
}

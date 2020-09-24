package faas

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Function struct {
	// Root on disk at which to find/create source and config files.
	Root string

	// Name of the Function.  If not provided, path derivation is attempted when
	// requried (such as for initialization).
	Name string

	// Namespace into which the Function is deployed on supported platforms.
	Namespace string

	// Runtime is the language plus context.  nodejs|go|quarkus|rust etc.
	Runtime string

	// Trigger of the Function.  http|events etc.
	Trigger string

	// Repository at which to store interstitial containers, in the form
	// [registry]/[user]. If omitted, "Image" must be provided.
	Repo string

	// Optional full OCI image tag in form:
	//   [registry]/[namespace]/[name]:[tag]
	// example:
	//   quay.io/alice/my.function.name
	// Registry is optional and is defaulted to DefaultRegistry
	// example:
	//   alice/my.function.name
	// If Image is provided, it overrides the default of concatenating
	// "Repo+Name:latest" to derive the Image.
	Image string

	// Builder represents the CNCF Buildpack builder image for a function
	Builder string
}

// NewFunction loads a Function from a path on disk. use .Initialized() to determine if
// the path contained an initialized Function.
// NewFunction creates a Function struct whose attributes are loaded from the
// configuraiton located at path.
func NewFunction(root string) (f Function, err error) {

	// Expand the passed root to its absolute path (default current dir)
	if root, err = filepath.Abs(root); err != nil {
		return
	}

	// Load a Config from the given absolute path
	c, err := newConfig(root)
	if err != nil {
		return
	}

	// Let's set Function name, if it is not already set
	if c.Name == "" {
		pathParts := strings.Split(strings.TrimRight(root, string(os.PathSeparator)), string(os.PathSeparator))
		c.Name = pathParts[len(pathParts)-1]
	}

	// set Function to the value of the config loaded from disk.
	f = fromConfig(c)

	// The only value not included in the config is the effective path on disk
	f.Root = root
	return
}

// WriteConfig writes this Function's configuration to disk.
func (f Function) WriteConfig() (err error) {
	return writeConfig(f)
}

// Initialized returns if the Function has been initialized.
// Any errors are considered failure (invalid or inaccessible root, config file, etc).
func (f Function) Initialized() bool {
	// Load the Function's configuration from disk and check if the (required) value Runtime is populated.
	c, err := newConfig(f.Root)
	if err != nil {
		return false
	}
	return c.Name != "" // TODO: use a dedicated initialized bool?
}

// DerivedImage returns the derived image name (OCI container tag) of the
// Function whose source is at root, with the default repository for when
// the image has to be calculated (derived).
// repository can be either with or without prefixed registry.
// The following are eqivalent due to the use of DefaultRegistry:
// repository:  docker.io/myname
//              myname
// A full image name consists of registry, repository, name and tag.
// in form [registry]/[repository]/[name]:[tag]
// example docker.io/alice/my.example.func:latest
// Default if not provided is --repository (a required global setting)
// followed by the provided (or derived) image name.
func DerivedImage(root, repository string) (image string, err error) {
	f, err := NewFunction(root)
	if err != nil {
		// an inability to load the Function means it is not yet initialized
		// We could try to be smart here and fall through to the Function name
		// deriviation logic, but that's likely to be confusing.  Better to
		// stay simple and say that derivation of Image depends on first having
		// the Function initialized.
		return
	}

	// If the Function has already had image populated, use this pre-calculated value.
	if f.Image != "" {
		image = f.Image
		return
	}

	// Repository is currently required until such time as we support
	// pushing to an implicitly-available in-cluster registry by default.
	if repository == "" {
		err = errors.New("Repository name is required.")
		return
	}

	// If the Function loaded, and there is not yet an Image set, then this is
	// the first build and no explicit image override was specified.  We should
	// therefore derive the image tag from the defined repository and name.
	// form:    [registry]/[user]/[function]:latest
	// example: quay.io/alice/my.function.name:latest
	repository = strings.Trim(repository, "/") // too defensive?
	repositoryTokens := strings.Split(repository, "/")
	if len(repositoryTokens) == 1 {
		image = DefaultRegistry + "/" + repository + "/" + f.Name
	} else if len(repositoryTokens) == 2 {
		image = repository + "/" + f.Name
	} else {
		err = fmt.Errorf("repository should be either 'namespace' or 'registry/namespace'")
	}

	// Explicitly append :latest.  We currently expect source control to drive
	// versioning, rather than rely on Docker Hub tags with explicit version
	// numbers, as is seen in many serverless solutions.  This will be updated
	// to branch name when we add source-driven canary/ bluegreen deployments.
	image = image + ":latest"
	return
}

// assertEmptyRoot ensures that the directory is empty enough to be used for
// initializing a new Function.
func assertEmptyRoot(path string) (err error) {
	// If there exists contentious files (congig files for instance), this Function may have already been initialized.
	files, err := contentiousFilesIn(path)
	if err != nil {
		return
	} else if len(files) > 0 {
		return fmt.Errorf("The chosen directory '%v' contains contentious files: %v.  Has the Service Function already been created?  Try either using a different directory, deleting the Function if it exists, or manually removing the files.", path, files)
	}

	// Ensure there are no non-hidden files, and again none of the aforementioned contentious files.
	empty, err := isEffectivelyEmpty(path)
	if err != nil {
		return
	} else if !empty {
		err = errors.New("The directory must be empty of visible files and recognized config files before it can be initialized.")
		return
	}
	return
}

// contentiousFiles are files which, if extant, preclude the creation of a
// Function rooted in the given directory.
var contentiousFiles = []string{
	ConfigFile,
}

// contentiousFilesIn the given directoy
func contentiousFilesIn(dir string) (contentious []string, err error) {
	files, err := ioutil.ReadDir(dir)
	for _, file := range files {
		for _, name := range contentiousFiles {
			if file.Name() == name {
				contentious = append(contentious, name)
			}
		}
	}
	return
}

// effectivelyEmpty directories are those which have no visible files
func isEffectivelyEmpty(dir string) (bool, error) {
	// Check for any non-hidden files
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return false, nil
		}
	}
	return true, nil
}

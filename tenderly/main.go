package tenderly

import (
	"github.com/linxGnu/grocksdb"
)

// IMPORTANT: Just to load grocksdb package
func main() {
	options := grocksdb.NewDefaultOptions()
	defer options.Destroy()
}

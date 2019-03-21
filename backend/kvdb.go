package backend

import (
	"fmt"

	"github.com/libopenstorage/openstorage/pkg/dbg"
	"github.com/portworx/kvdb"
	kvdbEtcdV3 "github.com/portworx/kvdb/etcd/v3"
)

func getKvdb(
	kvdbName string, // Use one of the kv store implementation names
	basePath string, // The path under which all the keys will be created by this kv instance
	discoveryEndpoints []string, // A list of kv store endpoints
	options map[string]string, // Options that need to be passed to the kv store
	panicHandler kvdb.FatalErrorCB, // A callback function to execute when the library needs to panic
) (kvdb.Kvdb, error) {

	kv, err := kvdb.New(
		kvdbName,
		basePath,
		discoveryEndpoints,
		options,
		panicHandler,
	)
	return kv, err

}

// A ...
type A struct {
	a1 string
	a2 int
}

func main() {

	// An example kvdb using etcd v3 as a key value store
	kv, err := getKvdb(
		kvdbEtcdV3.Name,
		"root/",
		[]string{"127.0.0.1:2379"},
		nil,
		dbg.Panicf,
	)
	if err != nil {
		fmt.Println("Failed to create a kvdb instance: ", err)
		return
	}

	// Put a key value pair foo=bar
	a := &A{"bar", 1}
	_, err = kv.Put("foo", &a, 0)
	if err != nil {
		fmt.Println("Failed to put a key in kvdb: ", err)
		return
	}

	// Get a key
	value := A{}
	_, err = kv.GetVal("foo", &value)
	if err != nil {
		fmt.Println("Failed to get a key from kvdb: ", err)
		return
	}
}

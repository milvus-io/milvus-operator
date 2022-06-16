package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/milvus-io/milvus-operator/pkg/util"
	"sigs.k8s.io/yaml"
)

func main() {
	srcPath := flag.String("s", "", "source yaml path, will overwrite the dst config")
	dstPath := flag.String("d", "", "destination yaml path, will be overwritten by the src config")
	flag.Parse()

	if *srcPath == "" || *dstPath == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
		return
	}

	src, err := readYaml(*srcPath)
	if err != nil {
		log.Fatal("read source yaml failed: ", err)
	}

	dst, err := readYaml(*dstPath)
	if err != nil {
		log.Fatal("read destination yaml failed: ", err)
	}
	util.MergeValues(dst, src)
	// adhoc to delete pulsar fields if not exist in src
	if src["pulsar"] == nil {
		delete(dst, "pulsar")
	}
	// adhoc to delete rocksmq fields if not exist in src
	if src["rocksmq"] == nil {
		delete(dst, "rocksmq")
	}
	// adhoc to delete kafka fields if not exist in src
	if src["kafka"] == nil {
		delete(dst, "kafka")
	}

	bs, err := yaml.Marshal(dst)
	if err != nil {
		log.Fatal("marshal failed: ", err)
	}

	if err := ioutil.WriteFile(*dstPath, bs, 0644); err != nil {
		log.Fatal("write failed: ", err)
	}
}

// readYaml
func readYaml(path string) (map[string]interface{}, error) {
	var data map[string]interface{}
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(bs, &data); err != nil {
		return nil, err
	}
	return data, nil
}

package connect

import (
	"bytes"
	_ "embed" // golint
	"sort"
	"strings"
	"text/template"
)

var (
	//go:embed extensions-list.tmpl
	extensionsListTemplate string
)

// helper struct to simplify extensions list template
type displayExtension struct {
	Product       Product
	Code          string
	Activated     bool
	Subextensions []displayExtension

	Indent     string
	ConnectCmd string
}

func GetExtensionsList() (string, error) {
	if !IsRegistered() {
		return "", ErrListExtensionsUnregistered
	}
	extensions, err := getExtensions()
	if err != nil {
		return "", err
	}
	activations, err := systemActivations()
	if err != nil {
		return "", err
	}
	return printExtensions(extensions, activations, isRootFSWritable())
}

func printExtensions(extensions []Product, activations map[string]Activation, rootFSWritable bool) (string, error) {
	t, err := template.New("extensions-list").Parse(extensionsListTemplate)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	cmd := "SUSEConnect"
	if !rootFSWritable {
		cmd = "transactional-update register"
	}
	err = t.Execute(&output, preformatExtensions(extensions, activations, cmd, 1))
	if err != nil {
		return "", err
	}
	return output.String(), nil
}

func preformatExtensions(extensions []Product, activations map[string]Activation, cmd string, level int) []displayExtension {
	var ret []displayExtension

	for _, e := range extensions {
		_, activated := activations[e.toTriplet()]
		ret = append(ret, displayExtension{
			Product:       e,
			Code:          e.toTriplet(),
			Activated:     activated,
			Subextensions: preformatExtensions(e.Extensions, activations, cmd, level+1),
			Indent:        strings.Repeat("    ", level),
			ConnectCmd:    cmd,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Product.FriendlyName < ret[j].Product.FriendlyName
	})

	return ret
}

func getExtensions() ([]Product, error) {
	base, err := baseProduct()
	if err != nil {
		return []Product{}, err
	}
	statuses, err := getStatuses()
	if err != nil {
		return []Product{}, err
	}

	if statuses[base.toTriplet()].Status != registered {
		return []Product{}, ErrListExtensionsUnregistered
	}

	remoteProductData, err := showProduct(base)
	if err != nil {
		return []Product{}, err
	}
	return remoteProductData.Extensions, nil
}

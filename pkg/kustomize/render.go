package kustomize

import (
	"bytes"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	transformimpl "sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/target"
)

// RenderOptions ...
type RenderOptions struct {
	Source string
	//	Out    io.Writer
}

// Render is used to render the kustomization
func Render(o RenderOptions) ([]byte, error) {
	fSys := fs.MakeFsOnDisk()
	uf := kunstruct.NewKunstructuredFactoryImpl()
	ptf := transformimpl.NewFactoryImpl()
	rf := resmap.NewFactory(resource.NewFactory(uf), ptf)
	v := validator.NewKustValidator()
	pluginCfg := plugins.DefaultPluginConfig()

	pl := plugins.NewLoader(pluginCfg, rf)

	loadRestrictor := loader.RestrictionRootOnly
	ldr, err := loader.NewLoader(loadRestrictor, v, o.Source, fSys)
	if err != nil {
		return nil, err
	}
	defer ldr.Cleanup()
	kt, err := target.NewKustTarget(ldr, rf, ptf, pl)
	if err != nil {
		return nil, err
	}
	m, err := kt.MakeCustomizedResMap()
	if err != nil {
		return nil, err
	}
	return m.AsYaml()
}

// GetLabels return labels
func GetLabels(yamlB []byte) (interface{}, error) {
	dec := yaml.NewDecoder(bytes.NewReader(yamlB))
	data := map[string]interface{}{}
	err := dec.Decode(data)

	return data["metadata"].(map[interface{}]interface{})["labels"], err
}

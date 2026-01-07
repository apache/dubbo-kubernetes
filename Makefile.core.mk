gen: generate-k8s-client tidy-go

clean: clean-k8s-client

applyconfiguration_gen = applyconfiguration-gen
kubetype_gen = kubetype-gen
deepcopy_gen = deepcopy-gen
client_gen = client-gen
lister_gen = lister-gen
informer_gen = informer-gen

kube_dubbo_source_packages = github.com/apache/dubbo-kubernetes/api/networking/v1alpha3

kube_base_output_package = client-go/pkg
kube_api_base_package = $(kube_base_output_package)/apis
kube_api_packages = github.com/apache/dubbo-kubernetes/$(kube_api_base_package)/networking/v1alpha3
kube_api_applyconfiguration_packages = $(kube_api_packages),k8s.io/apimachinery/pkg/apis/meta/v1
kube_clientset_package = $(kube_base_output_package)/clientset
kube_clientset_name = versioned
kube_listers_package = $(kube_base_output_package)/listers
kube_informers_package = $(kube_base_output_package)/informers
kube_applyconfiguration_package = $(kube_base_output_package)/applyconfiguration
kube_go_header_text = header.go.txt

empty:=
space := $(empty) $(empty)
comma := ,

ifeq ($(IN_BUILD_CONTAINER),1)
	# k8s code generators rely on GOPATH, using $GOPATH/src as the base package
	# directory.  Using --output-base . does not work, as that ends up generating
	# code into ./<package>, e.g. ./client-go/pkg/apis/...  To work
	# around this, we'll just let k8s generate the code where it wants and copy
	# back to where it should have been generated.
	move_generated=([ -d $(GOPATH)/src/$(kube_base_output_package)/ ] && cp -r $(GOPATH)/src/$(kube_base_output_package)/ client-go/ && rm -rf $(GOPATH)/src/$(kube_base_output_package)/) || true
else
	# nothing special for local builds
	move_generated=
endif

rename_generated_files=\
	cd client-go && find $(subst client-go/, $(empty), $(subst $(comma), $(space), $(kube_api_packages)) $(subst github.com/apache/dubbo-kubernetes/, $(empty), $(kube_clientset_package)) $(subst github.com/apache/dubbo-kubernetes/, $(empty), $(kube_listers_package)) $(subst github.com/apache/dubbo-kubernetes/, $(empty), $(kube_informers_package))) \
	-name '*.go' -and -not -name 'doc.go' -and -not -name '*.gen.go' -type f -exec sh -c 'mv "$$1" "$${1%.go}".gen.go' - '{}' \; || true

fixup_generated_files=\
	find client-go -name "*.deepcopy.gen.go" -type f -exec sed -i '' -e '/\*out = \*in/d' {} +

.PHONY: generate-k8s-client
generate-k8s-client:
	# generate kube api type wrappers for dubbo types
	@GODEBUG=gotypesalias=0 $(kubetype_gen) --input-dirs $(kube_dubbo_source_packages) --output-package $(kube_api_base_package) -h $(kube_go_header_text)
	@$(move_generated)
	# generate deepcopy for kube api types
	@$(deepcopy_gen) --input-dirs $(kube_api_packages) -O zz_generated.deepcopy  -h $(kube_go_header_text)
	# generate ssa for kube api types
	@$(applyconfiguration_gen) --input-dirs $(kube_api_applyconfiguration_packages) --output-package $(kube_applyconfiguration_package) -h $(kube_go_header_text)
	# generate clientsets for kube api types
	@$(client_gen) --clientset-name $(kube_clientset_name) --input-base "" --input  $(kube_api_packages) --output-package $(kube_clientset_package) -h $(kube_go_header_text) --apply-configuration-package $(kube_applyconfiguration_package)
	# generate listers for kube api types
	@$(lister_gen) --input-dirs $(kube_api_packages) --output-package $(kube_listers_package) -h $(kube_go_header_text)
	# generate informers for kube api types
	@$(informer_gen) --input-dirs $(kube_api_packages) --versioned-clientset-package $(kube_clientset_package)/$(kube_clientset_name) --listers-package $(kube_listers_package) --output-package $(kube_informers_package) -h $(kube_go_header_text)
	@$(move_generated)
	@$(rename_generated_files)
	@$(fixup_generated_files)

.PHONY: clean-k8s-client
clean-k8s-client:
    # remove generated code
	@rm -rf client-go/pkg

include Makefile.common.mk

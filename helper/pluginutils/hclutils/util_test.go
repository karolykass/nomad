package hclutils_test

import (
	"testing"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/nomad/drivers/docker"
	"github.com/hashicorp/nomad/helper/pluginutils/hclspecutils"
	"github.com/hashicorp/nomad/helper/pluginutils/hclutils"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/kr/pretty"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
	"github.com/ugorji/go/codec"
	"github.com/zclconf/go-cty/cty"
)

func hclConfigToInterface(t *testing.T, config string) interface{} {
	t.Helper()

	// Parse as we do in the jobspec parser
	root, err := hcl.Parse(config)
	if err != nil {
		t.Fatalf("failed to hcl parse the config: %v", err)
	}

	// Top-level item should be a list
	list, ok := root.Node.(*ast.ObjectList)
	if !ok {
		t.Fatalf("root should be an object")
	}

	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, list.Items[0]); err != nil {
		t.Fatalf("failed to decode object: %v", err)
	}

	var m2 map[string]interface{}
	if err := mapstructure.WeakDecode(m, &m2); err != nil {
		t.Fatalf("failed to weak decode object: %v", err)
	}

	return m2["config"]
}

func jsonConfigToInterface(t *testing.T, config string) interface{} {
	t.Helper()

	// Decode from json
	dec := codec.NewDecoderBytes([]byte(config), structs.JsonHandle)

	var m map[string]interface{}
	err := dec.Decode(&m)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	return m["Config"]
}

func TestParseHclInterface_Hcl(t *testing.T) {
	dockerDriver := new(docker.Driver)
	dockerSpec, err := dockerDriver.TaskConfigSchema()
	require.NoError(t, err)
	dockerDecSpec, diags := hclspecutils.Convert(dockerSpec)
	require.False(t, diags.HasErrors())

	vars := map[string]cty.Value{
		"NOMAD_ALLOC_INDEX": cty.NumberIntVal(2),
		"NOMAD_META_hello":  cty.StringVal("world"),
	}

	cases := []struct {
		name         string
		config       interface{}
		spec         hcldec.Spec
		vars         map[string]cty.Value
		expected     interface{}
		expectedType interface{}
	}{
		{
			name: "single string attr",
			config: hclConfigToInterface(t, `
			config {
				image = "redis:3.2"
			}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:   "redis:3.2",
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "single string attr json",
			config: jsonConfigToInterface(t, `
						{
							"Config": {
								"image": "redis:3.2"
			                }
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:   "redis:3.2",
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "number attr",
			config: hclConfigToInterface(t, `
						config {
							image = "redis:3.2"
							pids_limit  = 2
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:     "redis:3.2",
				PidsLimit: 2,
				Devices:   []docker.DockerDevice{},
				Mounts:    []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "number attr json",
			config: jsonConfigToInterface(t, `
						{
							"Config": {
								"image": "redis:3.2",
								"pids_limit": "2"
			                }
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:     "redis:3.2",
				PidsLimit: 2,
				Devices:   []docker.DockerDevice{},
				Mounts:    []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "number attr interpolated",
			config: hclConfigToInterface(t, `
						config {
							image = "redis:3.2"
							pids_limit  = "${2 + 2}"
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:     "redis:3.2",
				PidsLimit: 4,
				Devices:   []docker.DockerDevice{},
				Mounts:    []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "number attr interploated json",
			config: jsonConfigToInterface(t, `
						{
							"Config": {
								"image": "redis:3.2",
								"pids_limit": "${2 + 2}"
			                }
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:     "redis:3.2",
				PidsLimit: 4,
				Devices:   []docker.DockerDevice{},
				Mounts:    []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "multi attr",
			config: hclConfigToInterface(t, `
						config {
							image = "redis:3.2"
							args = ["foo", "bar"]
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:   "redis:3.2",
				Args:    []string{"foo", "bar"},
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "multi attr json",
			config: jsonConfigToInterface(t, `
						{
							"Config": {
								"image": "redis:3.2",
								"args": ["foo", "bar"]
			                }
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:   "redis:3.2",
				Args:    []string{"foo", "bar"},
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "multi attr variables",
			config: hclConfigToInterface(t, `
						config {
							image = "redis:3.2"
							args = ["${NOMAD_META_hello}", "${NOMAD_ALLOC_INDEX}"]
							pids_limit = "${NOMAD_ALLOC_INDEX + 2}"
						}`),
			spec: dockerDecSpec,
			vars: vars,
			expected: &docker.TaskConfig{
				Image:     "redis:3.2",
				Args:      []string{"world", "2"},
				PidsLimit: 4,
				Devices:   []docker.DockerDevice{},
				Mounts:    []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "multi attr variables json",
			config: jsonConfigToInterface(t, `
						{
							"Config": {
								"image": "redis:3.2",
								"args": ["foo", "bar"]
			                }
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:   "redis:3.2",
				Args:    []string{"foo", "bar"},
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "port_map",
			config: hclConfigToInterface(t, `
			config {
				image = "redis:3.2"
				port_map {
					foo = 1234
					bar = 5678
				}
			}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image: "redis:3.2",
				PortMap: map[string]int{
					"foo": 1234,
					"bar": 5678,
				},
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "port_map json",
			config: jsonConfigToInterface(t, `
							{
								"Config": {
									"image": "redis:3.2",
									"port_map": [{
										"foo": 1234,
										"bar": 5678
									}]
				                }
							}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image: "redis:3.2",
				PortMap: map[string]int{
					"foo": 1234,
					"bar": 5678,
				},
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "devices",
			config: hclConfigToInterface(t, `
						config {
							image = "redis:3.2"
							devices = [
								{
									host_path = "/dev/sda1"
									container_path = "/dev/xvdc"
									cgroup_permissions = "r"
								},
								{
									host_path = "/dev/sda2"
									container_path = "/dev/xvdd"
								}
							]
						}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image: "redis:3.2",
				Devices: []docker.DockerDevice{
					{
						HostPath:          "/dev/sda1",
						ContainerPath:     "/dev/xvdc",
						CgroupPermissions: "r",
					},
					{
						HostPath:      "/dev/sda2",
						ContainerPath: "/dev/xvdd",
					},
				},
				Mounts: []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "docker_logging",
			config: hclConfigToInterface(t, `
				config {
					image = "redis:3.2"
					network_mode = "host"
					dns_servers = ["169.254.1.1"]
					logging {
					    type = "syslog"
					    config {
						tag  = "driver-test"
					    }
					}
				}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image:       "redis:3.2",
				NetworkMode: "host",
				DNSServers:  []string{"169.254.1.1"},
				Logging: docker.DockerLogging{
					Type: "syslog",
					Config: map[string]string{
						"tag": "driver-test",
					},
				},
				Devices: []docker.DockerDevice{},
				Mounts:  []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
		{
			name: "docker_json",
			config: jsonConfigToInterface(t, `
					{
						"Config": {
							"image": "redis:3.2",
							"devices": [
								{
									"host_path": "/dev/sda1",
									"container_path": "/dev/xvdc",
									"cgroup_permissions": "r"
								},
								{
									"host_path": "/dev/sda2",
									"container_path": "/dev/xvdd"
								}
							]
				}
					}`),
			spec: dockerDecSpec,
			expected: &docker.TaskConfig{
				Image: "redis:3.2",
				Devices: []docker.DockerDevice{
					{
						HostPath:          "/dev/sda1",
						ContainerPath:     "/dev/xvdc",
						CgroupPermissions: "r",
					},
					{
						HostPath:      "/dev/sda2",
						ContainerPath: "/dev/xvdd",
					},
				},
				Mounts: []docker.DockerMount{},
			},
			expectedType: &docker.TaskConfig{},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Logf("Val: % #v", pretty.Formatter(c.config))
			// Parse the interface
			ctyValue, diag := hclutils.ParseHclInterface(c.config, c.spec, c.vars)
			if diag.HasErrors() {
				for _, err := range diag.Errs() {
					t.Error(err)
				}
				t.FailNow()
			}

			// Test encoding
			taskConfig := &drivers.TaskConfig{}
			require.NoError(t, taskConfig.EncodeDriverConfig(ctyValue))

			// Test decoding
			require.NoError(t, taskConfig.DecodeDriverConfig(c.expectedType))

			require.EqualValues(t, c.expected, c.expectedType)

		})
	}
}

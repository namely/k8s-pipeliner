package builder

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type KubecostTestSuite struct {
	suite.Suite
}

func TestKubecost(t *testing.T) {
	suite.Run(t, new(KubecostTestSuite))
}
func (s *KubecostTestSuite) Test_getKubecostSizing() {
	tests := map[string]struct {
		profile string
		want    []byte
		wantErr bool
	}{
		"development": {
			profile: "development",
		},
		"production": {
			profile: "production",
		},
		"high-availability": {
			profile: "high-availability",
		},
	}
	for name, t := range tests {
		s.Run(name, func() {
			bytes, err := getKubecostSizing(t.profile, kubecostDefaultWindow)
			s.Assert().NoError(err)
			s.Assert().NotEmpty(bytes)
		})
	}
}

func (s *KubecostTestSuite) Test_getRecommendedCPUAndRam() {
	content, err := ioutil.ReadFile("kubecost_response_example_test.text")
	s.Require().NoError(err)

	tests := map[string]struct {
		name string
		want map[string]map[string]requests
	}{
		"test hcm-web-public": {
			name: "hcm-web-public",
			want: map[string]map[string]requests{
				"hcm-web-public": {
					"hcm":{
						requestsCPU: 1.5,
						requestsRAM: 3,
					},
					"istio-proxy": {
						requestsCPU: 0.1,
						requestsRAM: 0.125,
					},
				},
			},
		},
	}
	for name, t := range tests {
		s.Run(name, func() {
			got := getRecommendedCPUAndRam(content)
			s.Assert().True(!reflect.DeepEqual(got[t.name], t.want))
		})
	}
}

func (s *KubecostTestSuite) Test_GetKubecostData() {
	tests := map[string]struct{
		expectedError error
	}{
		"happy path":{
		},
	}
	for name, _ := range tests {
		s.Run(name, func() {
			data, err := GetKubecostData()
			s.Assert().NoError(err)
			s.Assert().NotEmpty(data)
			s.Assert().Len(data, len(profiles))
		})
	}
}

package exclusion

import "testing"

func TestIsNamespaceExcluded(t *testing.T) {
	type args struct {
		exclusionList []string
		namespace     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Ignore all namespaces with openshift-* kube-* ",
			args{
				exclusionList: []string{"^(openshift|kube)-"},
				namespace:     "openshift-mynamespace",
			},
			true,
		},
		{
			"Ignore all namespaces",
			args{
				exclusionList: []string{"([^{}]*)", "something-else"},
				namespace:     "qwerty",
			},
			true,
		},
		{
			"Ignore specific namespace ",
			args{
				exclusionList: []string{"^(foo)-", "specific-namespace"},
				namespace:     "specific-namespace",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNamespaceExcluded(tt.args.exclusionList, tt.args.namespace); got != tt.want {
				t.Errorf("IsNamespaceExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}

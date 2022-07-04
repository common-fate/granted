package browsers

import "testing"

func Test_expandRegion(t *testing.T) {
	type args struct {
		region string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"<blank>", args{""}, "us-east-1", false},
		{"us-east-1", args{"us-east-1"}, "us-east-1", false},
		{"ue1", args{"ue1"}, "us-east-1", false},
		{"ue", args{"ue"}, "us-east-1", false},
		{"afs1", args{"afs1"}, "af-south-1", false},
		{"apse3", args{"apse3"}, "ap-southeast-3", false},
		{"cc", args{"cc"}, "ca-central-1", false},
		{"cc1", args{"cc1"}, "ca-central-1", false},
		{"ec1", args{"ec1"}, "eu-central-1", false},
		{"euc1", args{"euc1"}, "eu-central-1", false},
		{"ms1", args{"ms1"}, "me-south-1", false},
		{"se1", args{"se1"}, "sa-east-1", false},
		{"uge1", args{"uge1"}, "us-gov-east-1", false},
		{"cnn1", args{"cnn1"}, "cn-north-1", false},

		// Special cases
		{"mes1", args{"mes1"}, "me-south-1", false},
		{"use1", args{"use1"}, "us-east-1", false},

		{"???", args{"???"}, "", true}, // Completely invalid
		{"a", args{"a"}, "", true},     // Right major, too short
		{"ax", args{"ax"}, "", true},   // Right major, too short
		{"aee", args{"aee"}, "", true}, // Right major & minor, trailing crap
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandRegion(tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("expandRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}

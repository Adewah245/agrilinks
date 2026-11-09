package services

import "testing"

func TestRecipientAliases(t *testing.T) {
	for _, tc := range []struct {
		input, wallet string
		id            int
		phone         string
		invalid       bool
	}{
		{input: " agw-u-00000002-ngn ", wallet: "AGW-U-00000002-NGN"},
		{input: "AS1234567890", wallet: "AS1234567890"},
		{input: "42", id: 42},
		{input: "08012345678", phone: "2348012345678"},
		{input: "+234 801 234 5678", phone: "2348012345678"},
		{input: "8012345678", phone: "2348012345678"},
		{input: "1234567890", id: 1234567890, phone: "2341234567890"},
		{input: "", invalid: true}, {input: "0", invalid: true}, {input: "user 08012345678", invalid: true},
		{input: "2348012345678abc", invalid: true}, {input: "1 OR 1=1", invalid: true},
	} {
		wallet, id, phone, err := recipientAliases(tc.input)
		if (err != nil) != tc.invalid || wallet != tc.wallet || id != tc.id || phone != tc.phone {
			t.Fatalf("%q: %q %d %q %v", tc.input, wallet, id, phone, err)
		}
	}
}

func TestRecipientPhotoExcludesPrivateImages(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"/static/uploads/profiles/ada.jpg", "/static/uploads/profiles/ada.jpg"},
		{"/static/assets/images/un.jpeg", "/static/assets/images/un.jpeg"},
		{"/static/uploads/users/registration.jpg", ""}, {"/private/verification/7", ""},
		{"https://example.com/photo.jpg", ""}, {"//example.com/photo.jpg", ""},
		{"/static/uploads/profiles/../users/secret.jpg", ""},
		{"/static/uploads/profiles/%2e%2e/users/secret.jpg", ""},
		{"/static/uploads/profiles/ada.jpg?token=secret", ""},
		{"", ""},
	} {
		want := tc.want
		if want == "" {
			want = "/static/assets/images/un.jpeg"
		}
		if got := publicRecipientPhoto(tc.input); got != want {
			t.Fatalf("private photo returned: %q -> %q", tc.input, got)
		}
	}
}

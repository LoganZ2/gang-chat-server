package chat

import "testing"

func TestMentionMessageTargetsUser(t *testing.T) {
	target := mentionTarget{
		UserID: "user_1",
		UID:    "10000001",
		Labels: map[string]struct{}{
			"panel name": {},
			"loginname":  {},
		},
		Admin: false,
	}

	cases := []struct {
		name         string
		body         string
		mentionsJSON string
		target       mentionTarget
		want         bool
	}{
		{
			name:         "structured direct user",
			mentionsJSON: `[{"type":"user","user_id":"user_1"}]`,
			target:       target,
			want:         true,
		},
		{
			name:         "structured all",
			mentionsJSON: `[{"type":"all"}]`,
			target:       target,
			want:         true,
		},
		{
			name:         "structured admins ignores members",
			mentionsJSON: `[{"type":"admins"}]`,
			target:       target,
			want:         false,
		},
		{
			name:         "structured admins matches admins",
			mentionsJSON: `[{"type":"admins"}]`,
			target: mentionTarget{
				UserID: "admin_1",
				UID:    "10000002",
				Labels: map[string]struct{}{"admin": {}},
				Admin:  true,
			},
			want: true,
		},
		{
			name:   "text all",
			body:   "@所有人 hello",
			target: target,
			want:   true,
		},
		{
			name:   "text room display name",
			body:   "hi @Panel Name",
			target: target,
			want:   true,
		},
		{
			name:   "email is ignored",
			body:   "mail a@b.com",
			target: target,
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mentionMessageTargetsUser(
				tc.body,
				tc.mentionsJSON,
				tc.target,
			); got != tc.want {
				t.Fatalf("mentionMessageTargetsUser() = %v, want %v", got, tc.want)
			}
		})
	}
}

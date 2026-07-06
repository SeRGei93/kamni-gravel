package handler

import (
	"strings"
	"testing"
)

func TestParseChatMembersCSV(t *testing.T) {
	// BOM + заголовок + валидные/битые строки.
	csv := "\xEF\xBB\xBFuser_id,username,first_name,last_name,is_bot,is_deleted,role,joined_at\n" +
		"1,anna,Anna,R,0,0,member,2026-05-15T00:00:00+00:00\n" +
		"2,adm,Adm,,0,0,admin,\n" +
		"3,botacc,Bot,,1,0,member,\n" +
		"4,gone,Ghost,,0,1,member,\n" + // is_deleted → пропуск
		"notanumber,x,X,,0,0,member,\n" + // битый user_id → пропуск
		"6,creator,Creator,,0,0,creator,\n"

	members, skipped, err := parseChatMembersCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if skipped != 2 {
		t.Fatalf("skipped = %d, want 2", skipped)
	}
	if len(members) != 4 {
		t.Fatalf("members = %d, want 4: %+v", len(members), members)
	}

	byID := map[int64]bool{}
	for _, m := range members {
		byID[m.TelegramUserID] = m.IsAdmin
	}
	if _, ok := byID[4]; ok {
		t.Fatal("deleted account must be skipped")
	}
	if !byID[2] || !byID[6] {
		t.Fatal("admin and creator must map to IsAdmin=true")
	}
	if byID[1] {
		t.Fatal("plain member must be IsAdmin=false")
	}

	var bot bool
	for _, m := range members {
		if m.TelegramUserID == 3 {
			bot = m.IsBot
		}
	}
	if !bot {
		t.Fatal("is_bot=1 must map to IsBot=true")
	}
}

func TestParseChatMembersCSVEmpty(t *testing.T) {
	members, skipped, err := parseChatMembersCSV(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(members) != 0 || skipped != 0 {
		t.Fatalf("empty csv: members=%d skipped=%d, want 0/0", len(members), skipped)
	}
}

func TestParseChatMembersCSVHeaderOnly(t *testing.T) {
	members, _, err := parseChatMembersCSV(strings.NewReader("user_id,username,first_name,last_name,is_bot,is_deleted,role,joined_at\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("header-only csv: members=%d, want 0", len(members))
	}
}

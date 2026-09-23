package crossnotify

import "testing"

func TestDiff(t *testing.T) {
	prev := map[uint32]string{1: "Online on Switch", 2: "Playing Kirby on Ryujinx", 3: "Online on 3DS"}
	now := map[uint32]string{1: "Online on Switch", 2: "Playing a game on Ryujinx", 4: "Online on Eden"}
	connected := func(pid uint32) bool { return false }
	got := map[uint32]string{}
	for _, c := range Diff(prev, now, connected) {
		got[c.FriendPID] = c.Line
	}
	want := map[uint32]string{2: "Playing a game on Ryujinx", 3: "", 4: "Online on Eden"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for pid, line := range want {
		if l, ok := got[pid]; !ok || l != line {
			t.Errorf("pid %d: %q (%v), want %q", pid, l, ok, line)
		}
	}
	// A friend who left the foreign set because they connected here is not
	// sent offline.
	if d := Diff(map[uint32]string{9: "Online on Switch"}, nil, func(pid uint32) bool { return pid == 9 }); len(d) != 0 {
		t.Fatalf("connected friend sent offline: %v", d)
	}
}

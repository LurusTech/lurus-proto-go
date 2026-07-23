package events

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	eventsv1 "github.com/hanmahong5-arch/lurus-proto-go/events/v1"
)

// Machine enforcement for the SCHEMA_RULES.md rules that buf cannot check:
// R5 (PII tagging), R6 (past-tense NSID), R7 (domain budget), R8 (no floats),
// plus two-way consistency between Registry and the compiled descriptors.
// R1/R2 field evolution stays with buf breaking; R3/R4 stay with review.

// supportTypes are messages inside event packages that are not event payloads.
var supportTypes = map[protoreflect.FullName]bool{
	"lurus.events.lugo.v1.TransferRef": true,
}

// lurus.events.v1 holds the envelope and annotation machinery, not payloads.
func isInfraPackage(pkg protoreflect.FullName) bool { return pkg == "lurus.events.v1" }

func eventFiles(t *testing.T) []protoreflect.FileDescriptor {
	t.Helper()
	var out []protoreflect.FileDescriptor
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(string(fd.Package()), "lurus.events.") {
			out = append(out, fd)
		}
		return true
	})
	if len(out) < 4 { // envelope+annotations, tally, lugo, sync
		t.Fatalf("only %d lurus.events.* files registered — import wiring broken", len(out))
	}
	return out
}

func walkFields(md protoreflect.MessageDescriptor, f func(protoreflect.FieldDescriptor)) {
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		f(fields.Get(i))
	}
	nested := md.Messages()
	for i := 0; i < nested.Len(); i++ {
		walkFields(nested.Get(i), f)
	}
}

func TestRegistryMatchesDescriptorsBothWays(t *testing.T) {
	byFullName := map[protoreflect.FullName]string{}
	for nsid, ctor := range Registry {
		if !strings.HasPrefix(nsid, "cn.lurus.") {
			t.Errorf("NSID %q must start with cn.lurus.", nsid)
		}
		fn := ctor().ProtoReflect().Descriptor().FullName()
		if !strings.HasPrefix(string(fn), "lurus.events.") {
			t.Errorf("registry entry %q points at %s, outside lurus.events.*", nsid, fn)
		}
		if prev, dup := byFullName[fn]; dup {
			t.Errorf("message %s registered under both %q and %q", fn, prev, nsid)
		}
		byFullName[fn] = nsid
	}

	for _, fd := range eventFiles(t) {
		if isInfraPackage(fd.Package()) {
			continue
		}
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			md := msgs.Get(i)
			if supportTypes[md.FullName()] {
				continue
			}
			if _, ok := byFullName[md.FullName()]; !ok {
				t.Errorf("message %s exists but is not in Registry — register the NSID (and SCHEMA_RULES.md) or list it as a support type", md.FullName())
			}
		}
	}
}

func TestRegistryNew(t *testing.T) {
	msg, ok := New("cn.lurus.tally.stock.received")
	if !ok || msg == nil {
		t.Fatal("known event type not constructable")
	}
	if _, ok := New("cn.lurus.nope.nothing.happened"); ok {
		t.Fatal("unknown event type reported as known")
	}
}

func TestR6PastTenseNaming(t *testing.T) {
	// Heuristic: some token of the NSID's last segment ends in "ed" or is a
	// known irregular past participle. Extend the list before renaming a fact.
	irregular := map[string]bool{
		"written": true, "sent": true, "sold": true, "paid": true,
		"built": true, "set": true, "made": true, "done": true,
	}
	for nsid := range Registry {
		last := nsid[strings.LastIndex(nsid, ".")+1:]
		ok := false
		for _, tok := range strings.Split(last, "_") {
			if strings.HasSuffix(tok, "ed") || irregular[tok] {
				ok = true
			}
		}
		if !ok {
			t.Errorf("R6: %q last segment %q does not read as a past-tense fact", nsid, last)
		}
	}
}

func TestR7DomainBudget(t *testing.T) {
	perDomain := map[string]int{}
	for nsid := range Registry {
		perDomain[nsid[:strings.LastIndex(nsid, ".")]]++
	}
	for domain, n := range perDomain {
		if n > 5 {
			t.Errorf("R7: domain %s has %d event types, budget is 5 — 宁缺毋滥", domain, n)
		}
	}
}

func TestR5PIIFieldsAreTagged(t *testing.T) {
	piiTokens := map[string]bool{
		"name": true, "email": true, "phone": true, "mobile": true,
		"address": true, "passport": true, "birthday": true,
	}
	// Full field names allowed untagged despite matching a token (justify each).
	exempt := map[protoreflect.FullName]bool{}
	for _, fd := range eventFiles(t) {
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			walkFields(msgs.Get(i), func(field protoreflect.FieldDescriptor) {
				hit := false
				for _, tok := range strings.Split(string(field.Name()), "_") {
					if piiTokens[tok] {
						hit = true
					}
				}
				if !hit || exempt[field.FullName()] {
					return
				}
				opts, _ := field.Options().(*descriptorpb.FieldOptions)
				if opts == nil || !proto.GetExtension(opts, eventsv1.E_Pii).(bool) {
					t.Errorf("R5: field %s looks like PII but lacks (lurus.events.v1.pii) = true", field.FullName())
				}
			})
		}
	}
}

func TestR8NoFloatingPoint(t *testing.T) {
	for _, fd := range eventFiles(t) {
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			walkFields(msgs.Get(i), func(field protoreflect.FieldDescriptor) {
				if field.Kind() == protoreflect.FloatKind || field.Kind() == protoreflect.DoubleKind {
					t.Errorf("R8: field %s is %s — money is int64 fen, quantities are int64 smallest units", field.FullName(), field.Kind())
				}
			})
		}
	}
}

package arrowipc

import (
	"encoding/binary"
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
)

// These slots are defined in Arrow format/File.fbs and Schema.fbs. Validate
// vectors before Arrow's decoder allocates slices from their lengths. All
// arithmetic on untrusted offsets is widened before addition so it cannot wrap.
func offsetTable(parent flatbuffers.Table, slot flatbuffers.VOffsetT) (flatbuffers.Table, error) {
	o := parent.Offset(slot)
	if o == 0 {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: missing metadata table at slot %d", slot)
	}
	return tableAt(parent.Bytes, uint64(parent.Pos)+uint64(o))
}
func tableAt(data []byte, slot uint64) (flatbuffers.Table, error) {
	if slot+4 > uint64(len(data)) {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: metadata table offset out of bounds")
	}
	target := slot + uint64(binary.LittleEndian.Uint32(data[slot:slot+4]))
	if target <= slot || target+4 > uint64(len(data)) {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: metadata table offset must point forward inside the metadata buffer")
	}
	return flatbuffers.Table{Bytes: data, Pos: flatbuffers.UOffsetT(target)}, nil
}
func boundedVector(tab flatbuffers.Table, slot flatbuffers.VOffsetT, width uint64) (uint64, int, error) {
	o := tab.Offset(slot)
	if o == 0 {
		return 0, 0, nil
	}
	at := uint64(tab.Pos) + uint64(o)
	if at+4 > uint64(len(tab.Bytes)) {
		return 0, 0, fmt.Errorf("arrowipc: metadata vector offset out of bounds")
	}
	lengthAt := at + uint64(binary.LittleEndian.Uint32(tab.Bytes[at:at+4]))
	if lengthAt <= at || lengthAt+4 > uint64(len(tab.Bytes)) {
		return 0, 0, fmt.Errorf("arrowipc: invalid metadata vector offset")
	}
	start := lengthAt + 4
	n := uint64(binary.LittleEndian.Uint32(tab.Bytes[lengthAt:start]))
	if n > (uint64(len(tab.Bytes))-start)/width {
		return 0, 0, fmt.Errorf("arrowipc: metadata vector length %d exceeds its buffer", n)
	}
	return start, int(n), nil
}
func validateSchemaFooter(footer flatbuffers.Table) error {
	schema, err := offsetTable(footer, 6)
	if err != nil {
		return err
	}
	// Repeated references must not expand a tiny footer into an unbounded tree.
	budget := len(footer.Bytes) / 4
	stringsBudget := int64(maxFooterSize)
	if err := validateKeyValueStrings(footer, 12, &stringsBudget); err != nil {
		return err
	}
	return validateFieldVector(schema, 6, 8, 0, &budget, &stringsBudget)
}
func validateFieldVector(tab flatbuffers.Table, fieldsSlot, metadataSlot flatbuffers.VOffsetT, depth int, budget *int, stringsBudget *int64) error {
	if depth > 64 {
		return fmt.Errorf("arrowipc: schema nesting exceeds 64 levels")
	}
	start, n, err := boundedVector(tab, fieldsSlot, 4)
	if err != nil {
		return err
	}
	if n > *budget {
		return fmt.Errorf("arrowipc: expanded schema exceeds footer bounds")
	}
	*budget -= n
	if _, metaCount, err := boundedVector(tab, metadataSlot, 4); err != nil {
		return err
	} else if metaCount > *budget {
		return fmt.Errorf("arrowipc: expanded schema metadata exceeds footer bounds")
	} else {
		*budget -= metaCount
	}
	if err := validateKeyValueStrings(tab, metadataSlot, stringsBudget); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		field, err := tableAt(tab.Bytes, start+uint64(i)*4)
		if err != nil {
			return err
		}
		if err := chargeMetadataString(field, 4, stringsBudget); err != nil {
			return err
		}
		// Schema.fbs Type.Timestamp is 10; its timezone is a copied string.
		if o := field.Offset(6); o != 0 && field.GetByte(field.Pos+flatbuffers.UOffsetT(o)) == 10 {
			timestamp, err := offsetTable(field, 8)
			if err != nil {
				return err
			}
			if err := chargeMetadataString(timestamp, 6, stringsBudget); err != nil {
				return err
			}
		}
		if err := validateFieldVector(field, 14, 16, depth+1, budget, stringsBudget); err != nil {
			return err
		}
	}
	return nil
}

// Arrow copies metadata strings, even when multiple offsets share wire bytes.
// Charge each occurrence before decoding, without allocating those strings.
func chargeMetadataString(tab flatbuffers.Table, slot flatbuffers.VOffsetT, budget *int64) error {
	_, n, err := boundedVector(tab, slot, 1)
	if err != nil {
		return err
	}
	if int64(n) > *budget {
		return fmt.Errorf("arrowipc: decoded metadata strings exceed %d bytes", maxFooterSize)
	}
	*budget -= int64(n)
	return nil
}

func validateKeyValueStrings(tab flatbuffers.Table, slot flatbuffers.VOffsetT, budget *int64) error {
	start, n, err := boundedVector(tab, slot, 4)
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		entry, err := tableAt(tab.Bytes, start+uint64(i)*4)
		if err != nil {
			return err
		}
		for _, textSlot := range []flatbuffers.VOffsetT{4, 6} {
			if err := chargeMetadataString(entry, textSlot, budget); err != nil {
				return err
			}
		}
	}
	return nil
}

package normalize

// KeyBuilder assembles tool+":"+canonical-args into the byte key
// ObserveBytes hashes internally, so no separate fingerprint step is
// needed. It reuses one buffer since exactly one goroutine calls Build.
type KeyBuilder struct {
	buf []byte
}

// NewKeyBuilder returns a ready-to-use KeyBuilder.
func NewKeyBuilder() *KeyBuilder {
	return &KeyBuilder{buf: make([]byte, 0, 256)}
}

// Build returns toolName+":"+canonicalArgs. The result aliases an
// internal buffer valid only until the next call — copy it
// (string(key)) to retain it longer.
func (b *KeyBuilder) Build(toolName string, canonicalArgs []byte) []byte {
	b.buf = b.buf[:0]
	b.buf = append(b.buf, toolName...)
	b.buf = append(b.buf, ':')
	b.buf = append(b.buf, canonicalArgs...)
	return b.buf
}

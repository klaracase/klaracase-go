package klaracase

// resetDeprecationState forgets which endpoints already raised a deprecation notice. The guard
// is process wide in production, so every test that looks at the handler calls this first.
func resetDeprecationState() {
	deprecationSeen.Range(func(k, _ any) bool {
		deprecationSeen.Delete(k)
		return true
	})
}

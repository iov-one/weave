package orm

func (m VersionedIDRef) NextVersion() VersionedIDRef {
	return VersionedIDRef{ID: m.ID, Version: m.Version + 1}
}

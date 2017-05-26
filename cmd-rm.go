package hush

func CmdRm(tree *Tree, paths []Path) error {
	n := tree.Delete(paths)
	if n > 0 {
		return tree.Save()
	}
	return nil
}

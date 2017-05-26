package hush

func CmdRm(tree *Tree, p Path) error {
	n := tree.Delete(p)
	if n > 0 {
		return tree.Save()
	}
	return nil
}

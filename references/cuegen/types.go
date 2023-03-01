/*
Copyright 2023 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cuegen

import (
	"fmt"
	goast "go/ast"
	gotypes "go/types"

	"golang.org/x/tools/go/packages"
)

type typeInfo map[gotypes.Type]*goast.StructType

func getTypeInfo(p *packages.Package) typeInfo {
	m := make(typeInfo)

	for _, f := range p.Syntax {
		goast.Inspect(f, func(n goast.Node) bool {
			// record all struct types
			if t, ok := n.(*goast.StructType); ok {
				m[p.TypesInfo.TypeOf(t)] = t
			}
			return true
		})
	}

	return m
}

func supportedType(stack []gotypes.Type, t gotypes.Type) error {
	// we expand structures recursively, so we can't support recursive types
	for _, t0 := range stack {
		if t0 == t {
			return fmt.Errorf("recursive type %s", t)
		}
	}
	stack = append(stack, t)

	t = t.Underlying()
	switch x := t.(type) {
	case *gotypes.Basic:
		if x.String() != "invalid type" {
			return nil
		}
		return fmt.Errorf("unsupported type %s", t)
	case *gotypes.Named:
		return nil
	case *gotypes.Pointer:
		return supportedType(stack, x.Elem())
	case *gotypes.Slice:
		return supportedType(stack, x.Elem())
	case *gotypes.Array:
		return supportedType(stack, x.Elem())
	case *gotypes.Map:
		if b, ok := x.Key().Underlying().(*gotypes.Basic); !ok || b.Kind() != gotypes.String {
			return fmt.Errorf("unsupported map key type %s of %s", x.Key(), t)
		}
		return supportedType(stack, x.Elem())
	case *gotypes.Struct:
		// Eliminate structs with fields for which all fields are filtered.
		if x.NumFields() == 0 {
			return nil
		}
		for i := 0; i < x.NumFields(); i++ {
			f := x.Field(i)
			if f.Exported() {
				if err := supportedType(stack, f.Type()); err != nil {
					return err
				}
			}
		}
		return nil
	case *gotypes.Interface:
		return nil
	}
	return fmt.Errorf("unsupported type %s", t)
}

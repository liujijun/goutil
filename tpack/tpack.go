package tpack

import (
	"reflect"
	"unsafe"
)

// U go underlying type data
type U struct {
	typPtr uintptr
	ptr    unsafe.Pointer
	_iPtr  unsafe.Pointer // avoid being GC
}

// Unpack unpacks i to go underlying type data.
func Unpack(i interface{}) U {
	return newT(unsafe.Pointer(&i))
}

// From gets go underlying type data from reflect.Value.
func From(v reflect.Value) U {
	return newT(unsafe.Pointer(&v))
}

func newT(iPtr unsafe.Pointer) U {
	return U{
		typPtr: *(*uintptr)(iPtr),
		ptr:    pointerElem(unsafe.Pointer(uintptr(iPtr) + ptrOffset)),
		_iPtr:  iPtr,
	}
}

// RuntimeTypeID returns the underlying type ID in current runtime from reflect.Type.
// NOTE:
//  *A and A returns the same runtime type ID;
//  It is 10 times performance of t.String().
func RuntimeTypeID(t reflect.Type) int32 {
	typPtr := uintptrElem(uintptr(unsafe.Pointer(&t)) + ptrOffset)
	return *(*int32)(unsafe.Pointer(typPtr + rtypeStrOffset))
}

// RuntimeTypeID gets the underlying type ID in current runtime.
// NOTE:
//  *A and A gets the same runtime type ID;
//  It is 10 times performance of reflect.TypeOf(i).String().
func (u U) RuntimeTypeID() int32 {
	return *(*int32)(unsafe.Pointer(u.typPtr + rtypeStrOffset))
}

// Kind gets the reflect.Kind fastly.
func (u U) Kind() reflect.Kind {
	return kind(u.typPtr)
}

// Elem returns the U that the interface i contains
// or that the pointer i points to.
func (u U) Elem() U {
	k := u.Kind()
	if k == reflect.Interface {
		return newT(u.ptr)
	}
	if k != reflect.Ptr {
		return u
	}
	var has bool
	k, u.typPtr, has = ptrUnderlying(k, u.typPtr)
	if !has {
		return u
	}
	if k == reflect.Ptr || k == reflect.Interface {
		u.ptr = pointerElem(u.ptr)
	}
	return u
}

// UnderlyingElem returns the underlying U that the interface i contains
// or that the pointer i points to.
func (u U) UnderlyingElem() U {
	for r := u.Elem(); r != u; r = u.Elem() {
		u = r
	}
	return u
}

// Pointer gets the pointer of i.
// NOTE:
//  *T and T, gets diffrent pointer
func (u U) Pointer() uintptr {
	k := u.Kind()
	switch k {
	case reflect.Invalid:
		return 0
	case reflect.Func:
		return uintptr(*(*unsafe.Pointer)(u.ptr))
	case reflect.Slice:
		return uintptrElem(uintptr(u.ptr)) + sliceDataOffset
	default:
		return uintptr(u.ptr)
	}
}

func ptrUnderlying(k reflect.Kind, typPtr uintptr) (reflect.Kind, uintptr, bool) {
	typPtr2 := uintptrElem(typPtr + elemOffset)
	k2 := kind(typPtr2)
	if k2 == reflect.Invalid {
		return k, typPtr, false
	}
	return k2, typPtr2, true
}

func kind(typPtr uintptr) reflect.Kind {
	if unsafe.Pointer(typPtr) == nil {
		return reflect.Invalid
	}
	k := *(*uint8)(unsafe.Pointer(typPtr + kindOffset))
	return reflect.Kind(k & kindMask)
}

func uintptrElem(ptr uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(ptr))
}

func pointerElem(p unsafe.Pointer) unsafe.Pointer {
	return *(*unsafe.Pointer)(p)
}

var (
	ptrOffset = func() uintptr {
		return unsafe.Offsetof(e.word)
	}()
	rtypeStrOffset = func() uintptr {
		return unsafe.Offsetof(e.typ.str)
	}()
	kindOffset = func() uintptr {
		return unsafe.Offsetof(e.typ.kind)
	}()
	elemOffset = func() uintptr {
		return unsafe.Offsetof(new(ptrType).elem)
	}()
	sliceLenOffset = func() uintptr {
		return unsafe.Offsetof(new(reflect.SliceHeader).Len)
	}()
	sliceDataOffset = func() uintptr {
		return unsafe.Offsetof(new(reflect.SliceHeader).Data)
	}()
	e = emptyInterface{typ: new(rtype)}
)

const (
	kindMask = (1 << 5) - 1
)

type (
	// reflectValue struct {
	// 	typ *rtype
	// 	ptr unsafe.Pointer
	// 	flag
	// }
	emptyInterface struct {
		typ  *rtype
		word unsafe.Pointer
	}
	rtype struct {
		size       uintptr
		ptrdata    uintptr  // number of bytes in the type that can contain pointers
		hash       uint32   // hash of type; avoids computation in hash tables
		tflag      tflag    // extra type information flags
		align      uint8    // alignment of variable with this type
		fieldAlign uint8    // alignment of struct field with this type
		kind       uint8    // enumeration for C
		alg        *typeAlg // algorithm table
		gcdata     *byte    // garbage collection data
		str        nameOff  // string form
		ptrToThis  typeOff  // type for pointer to this type, may be zero
	}
	ptrType struct {
		rtype
		elem *rtype // pointer element (pointed at) type
	}
	typeAlg struct {
		hash  func(unsafe.Pointer, uintptr) uintptr
		equal func(unsafe.Pointer, unsafe.Pointer) bool
	}
	nameOff int32 // offset to a name
	typeOff int32 // offset to an *rtype
	flag    uintptr
	tflag   uint8
)
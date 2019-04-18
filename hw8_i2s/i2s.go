package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	v := reflect.ValueOf(data)   // get reflection of input data
	vOut := reflect.ValueOf(out) // get reflection of output data

	if vOut.Kind() != reflect.Ptr {
		return fmt.Errorf("Output parameter is %v. Should be a pointer", vOut.Kind())
	}

	// check that input variables is
	switch v.Type().Kind() {
	case reflect.Map:
		// go trought each pair of the input map
		for _, k := range v.MapKeys() {
			// output field
			fOut := vOut.Elem().FieldByName(k.String())
			// select appropriate type of input map field
			switch reflect.TypeOf(v.MapIndex(k).Interface()).Kind() {
			case reflect.Float64:
				//check that output has the same type
				if fOut.Type().String() != "int" {
					return fmt.Errorf("Type of output %v field is not match type of input field. Type %v <> %v", k.String(), fOut.Type().String(), "int")
				}
				// set vOut
				fOut.SetInt(int64(v.MapIndex(k).Interface().(float64)))
			case reflect.String:
				if fOut.Type().String() != "string" {
					return fmt.Errorf("Type of output %v field is not match type of input field. Type %v <> %v", k.String(), fOut.Type().String(), "string")
				}
				fOut.SetString(v.MapIndex(k).Interface().(string))
			case reflect.Bool:
				if fOut.Type().String() != "bool" {
					return fmt.Errorf("Type of output %v field is not match type of input field. Type %v <> %v", k.String(), fOut.Type().String(), "bool")
				}
				fOut.SetBool(v.MapIndex(k).Interface().(bool))
			case reflect.Map:
				if fOut.Type().Kind().String() != "struct" {
					return fmt.Errorf("Type of output %v field is not match type of input field. Type %v <> %v", k.String(), fOut.Type().Kind().String(), "map")
				}
				// create a new struct element
				outVal := reflect.New(fOut.Type()).Elem()
				// recursion with a submap and a new struct element
				if err := i2s(v.MapIndex(k).Interface(), outVal.Addr().Interface()); err != nil {
					return err
				}
				fOut.Set(outVal)
			case reflect.Slice:
				if err := i2s(v.MapIndex(k).Interface(), fOut.Addr().Interface()); err != nil {
					return err
				}
			}
		}
	case reflect.Slice:
		if vOut.Elem().Kind() != reflect.Slice {
			return fmt.Errorf("Type of output %v field is not match type of input field. Type %v <> %v", vOut.String(), vOut.Elem().Kind(), "slice")
		}
		inputSlice := reflect.ValueOf(v.Interface())
		// prepare output slice
		outputSlice := reflect.MakeSlice(vOut.Elem().Type(), inputSlice.Len(), inputSlice.Len())
		// for each element of the slice call a recursion function
		for i := 0; i < inputSlice.Len(); i++ {
			if err := i2s(inputSlice.Index(i).Interface(), outputSlice.Index(i).Addr().Interface()); err != nil {
				return err
			}
		}
		vOut.Elem().Set(outputSlice)
	}

	return nil
}

package jsoniter

import (
	"encoding/json"
	"github.com/json-iterator/go/require"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

func Test_customize_type_decoder(t *testing.T) {
	RegisterTypeDecoderFunc("time.Time", func(ptr unsafe.Pointer, iter *Iterator) {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", iter.ReadString(), time.UTC)
		if err != nil {
			iter.Error = err
			return
		}
		*((*time.Time)(ptr)) = t
	})
	defer ConfigDefault.cleanDecoders()
	val := time.Time{}
	err := Unmarshal([]byte(`"2016-12-05 08:43:28"`), &val)
	if err != nil {
		t.Fatal(err)
	}
	year, month, day := val.Date()
	if year != 2016 || month != 12 || day != 5 {
		t.Fatal(val)
	}
}

func Test_customize_type_encoder(t *testing.T) {
	should := require.New(t)
	RegisterTypeEncoderFunc("time.Time", func(ptr unsafe.Pointer, stream *Stream) {
		t := *((*time.Time)(ptr))
		stream.WriteString(t.UTC().Format("2006-01-02 15:04:05"))
	}, nil)
	defer ConfigDefault.cleanEncoders()
	val := time.Unix(0, 0)
	str, err := MarshalToString(val)
	should.Nil(err)
	should.Equal(`"1970-01-01 00:00:00"`, str)
}

func Test_customize_byte_array_encoder(t *testing.T) {
	ConfigDefault.cleanEncoders()
	should := require.New(t)
	RegisterTypeEncoderFunc("[]uint8", func(ptr unsafe.Pointer, stream *Stream) {
		t := *((*[]byte)(ptr))
		stream.WriteString(string(t))
	}, nil)
	defer ConfigDefault.cleanEncoders()
	val := []byte("abc")
	str, err := MarshalToString(val)
	should.Nil(err)
	should.Equal(`"abc"`, str)
}

func Test_customize_float_marshal(t *testing.T) {
	should := require.New(t)
	json := Config{MarshalFloatWith6Digits: true}.Froze()
	str, err := json.MarshalToString(float32(1.23456789))
	should.Nil(err)
	should.Equal("1.234568", str)
}

type Tom struct {
	field1 string
}

func Test_customize_field_decoder(t *testing.T) {
	RegisterFieldDecoderFunc("jsoniter.Tom", "field1", func(ptr unsafe.Pointer, iter *Iterator) {
		*((*string)(ptr)) = strconv.Itoa(iter.ReadInt())
	})
	defer ConfigDefault.cleanDecoders()
	tom := Tom{}
	err := Unmarshal([]byte(`{"field1": 100}`), &tom)
	if err != nil {
		t.Fatal(err)
	}
}

type TestObject1 struct {
	field1 string
}

type testExtension struct {
	DummyExtension
}

func (extension *testExtension) UpdateStructDescriptor(structDescriptor *StructDescriptor) {
	if structDescriptor.Type.String() != "jsoniter.TestObject1" {
		return
	}
	binding := structDescriptor.GetField("field1")
	binding.Encoder = &funcEncoder{fun: func(ptr unsafe.Pointer, stream *Stream) {
		str := *((*string)(ptr))
		val, _ := strconv.Atoi(str)
		stream.WriteInt(val)
	}}
	binding.Decoder = &funcDecoder{func(ptr unsafe.Pointer, iter *Iterator) {
		*((*string)(ptr)) = strconv.Itoa(iter.ReadInt())
	}}
	binding.ToNames = []string{"field-1"}
	binding.FromNames = []string{"field-1"}
}

func Test_customize_field_by_extension(t *testing.T) {
	should := require.New(t)
	RegisterExtension(&testExtension{})
	obj := TestObject1{}
	err := UnmarshalFromString(`{"field-1": 100}`, &obj)
	should.Nil(err)
	should.Equal("100", obj.field1)
	str, err := MarshalToString(obj)
	should.Nil(err)
	should.Equal(`{"field-1":100}`, str)
}

//func Test_unexported_fields(t *testing.T) {
//	jsoniter := Config{SupportUnexportedStructFields: true}.Froze()
//	should := require.New(t)
//	type TestObject struct {
//		field1 string
//		field2 string `json:"field-2"`
//	}
//	obj := TestObject{}
//	obj.field1 = "hello"
//	should.Nil(jsoniter.UnmarshalFromString(`{}`, &obj))
//	should.Equal("hello", obj.field1)
//	should.Nil(jsoniter.UnmarshalFromString(`{"field1": "world", "field-2": "abc"}`, &obj))
//	should.Equal("world", obj.field1)
//	should.Equal("abc", obj.field2)
//	str, err := jsoniter.MarshalToString(obj)
//	should.Nil(err)
//	should.Contains(str, `"field-2":"abc"`)
//}

type timeImplementedMarshaler time.Time

func (obj timeImplementedMarshaler) MarshalJSON() ([]byte, error) {
	seconds := time.Time(obj).Unix()
	return []byte(strconv.FormatInt(seconds, 10)), nil
}

func Test_marshaler(t *testing.T) {
	type TestObject struct {
		Field timeImplementedMarshaler
	}
	should := require.New(t)
	val := timeImplementedMarshaler(time.Unix(123, 0))
	obj := TestObject{val}
	bytes, err := json.Marshal(obj)
	should.Nil(err)
	should.Equal(`{"Field":123}`, string(bytes))
	str, err := MarshalToString(obj)
	should.Nil(err)
	should.Equal(`{"Field":123}`, str)
}

func Test_marshaler_and_encoder(t *testing.T) {
	type TestObject struct {
		Field *timeImplementedMarshaler
	}
	ConfigDefault.cleanEncoders()
	should := require.New(t)
	RegisterTypeEncoderFunc("jsoniter.timeImplementedMarshaler", func(ptr unsafe.Pointer, stream *Stream) {
		stream.WriteString("hello from encoder")
	}, nil)
	val := timeImplementedMarshaler(time.Unix(123, 0))
	obj := TestObject{&val}
	bytes, err := json.Marshal(obj)
	should.Nil(err)
	should.Equal(`{"Field":123}`, string(bytes))
	str, err := MarshalToString(obj)
	should.Nil(err)
	should.Equal(`{"Field":"hello from encoder"}`, str)
}

type ObjectImplementedUnmarshaler int

func (obj *ObjectImplementedUnmarshaler) UnmarshalJSON(s []byte) error {
	val, _ := strconv.ParseInt(string(s[1:len(s)-1]), 10, 64)
	*obj = ObjectImplementedUnmarshaler(val)
	return nil
}

func Test_unmarshaler(t *testing.T) {
	should := require.New(t)
	var obj ObjectImplementedUnmarshaler
	err := json.Unmarshal([]byte(`   "100" `), &obj)
	should.Nil(err)
	should.Equal(100, int(obj))
	iter := ParseString(ConfigDefault, `   "100" `)
	iter.ReadVal(&obj)
	should.Nil(err)
	should.Equal(100, int(obj))
}

func Test_unmarshaler_and_decoder(t *testing.T) {
	type TestObject struct {
		Field  *ObjectImplementedUnmarshaler
		Field2 string
	}
	ConfigDefault.cleanDecoders()
	should := require.New(t)
	RegisterTypeDecoderFunc("jsoniter.ObjectImplementedUnmarshaler", func(ptr unsafe.Pointer, iter *Iterator) {
		*(*ObjectImplementedUnmarshaler)(ptr) = 10
		iter.Skip()
	})
	obj := TestObject{}
	val := ObjectImplementedUnmarshaler(0)
	obj.Field = &val
	err := json.Unmarshal([]byte(`{"Field":"100"}`), &obj)
	should.Nil(err)
	should.Equal(100, int(*obj.Field))
	err = Unmarshal([]byte(`{"Field":"100"}`), &obj)
	should.Nil(err)
	should.Equal(10, int(*obj.Field))
}

type tmString string
type tmStruct struct {
	String tmString
}

func (s tmStruct) MarshalJSON() ([]byte, error) {
	var b []byte
	b = append(b, '"')
	b = append(b, s.String...)
	b = append(b, '"')
	return b, nil
}

func Test_marshaler_on_struct(t *testing.T) {
	fixed := tmStruct{"hello"}
	//json.Marshal(fixed)
	Marshal(fixed)
}

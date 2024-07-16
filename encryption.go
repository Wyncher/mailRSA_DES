package main

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strconv"
)

type encrypt_struct struct {
	key_des               []byte
	iv_des                []byte
	body_encrypt          []byte
	des_encrypt_key       []byte
	receiverPublicKey_rsa rsa.PublicKey
	senderPrivateKey_rsa  rsa.PrivateKey
	input_str             string
	signature             []byte
	from                  string
	to                    string
}

type decrypt_struct struct {
	key_des             []byte
	iv_des              []byte
	cryptoText_des      []byte
	body_decrypt        []byte
	PrivateKey_rsa      rsa.PrivateKey
	senderPublicKey_rsa rsa.PublicKey
	des_encrypt_key     []byte
	signature           []byte
}

// DES
func DesEncryption(key, iv, plainText []byte) ([]byte, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData := PKCS5Padding(plainText, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv)
	cryted := make([]byte, len(origData))
	blockMode.CryptBlocks(cryted, origData)
	return cryted, nil
}
func DesDecryption(key, iv, cipherText []byte) ([]byte, error) {

	block, err := des.NewCipher(key)

	if err != nil {
		return nil, err
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(cipherText))
	blockMode.CryptBlocks(origData, cipherText)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}
func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}
func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}
func encryptdes(originalText string, encrypt_struct encrypt_struct) encrypt_struct {
	//des start

	encrypt_struct.input_str = originalText

	mytext := []byte(originalText)
	encrypt_struct.key_des = []byte(GenerateRandomString(8))
	encrypt_struct.iv_des = []byte(GenerateRandomString(8))
	cryptoText, _ := DesEncryption(encrypt_struct.key_des, encrypt_struct.iv_des, mytext)
	//fmt.Println("шифрофраза " + string(cryptoText))
	//fmt.Println("ключ симметричного алгоритима Des   " + string(encrypt_struct.key_des[:]))
	encrypt_struct.body_encrypt = cryptoText
	return encrypt_struct
	//des end
}
func decryptdes(decrypt_struct decrypt_struct) decrypt_struct {
	//des start
	decryptedText, _ := DesDecryption(decrypt_struct.key_des, decrypt_struct.iv_des, decrypt_struct.cryptoText_des)
	decrypt_struct.body_decrypt = decryptedText
	//fmt.Println("ключ симметричного алгоритима Des   " + string(decrypt_struct.key_des[:]))

	return decrypt_struct
	//des end
}

// RSA
func EncryptOAEP(encrypt_struct encrypt_struct) encrypt_struct {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &encrypt_struct.receiverPublicKey_rsa, encrypt_struct.key_des[:], label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from encryption: %s\n", err)
	}
	encrypt_struct.des_encrypt_key = ciphertext
	//base64.StdEncoding.EncodeToString(ciphertext)
	return encrypt_struct
}
func DecryptOAEP(decrypt_struct decrypt_struct) decrypt_struct {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, &decrypt_struct.PrivateKey_rsa, decrypt_struct.des_encrypt_key, label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from decryption: %s\n", err)
	}
	fmt.Printf("Plaintext: %s\n", string(plaintext))
	decrypt_struct.key_des = plaintext
	return decrypt_struct
}
func savePublicPEMKey(fileName string, pubkey rsa.PublicKey) {
	asn1Bytes, err := asn1.Marshal(pubkey)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	pemfile, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
	defer pemfile.Close()
	err = pem.Encode(pemfile, pemkey)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
func savePKCS8RSAPEMKey(fName string, key *rsa.PrivateKey) {
	outFile, err := os.Create(fName)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
	defer outFile.Close()
	//converts a private key to ASN.1 DER encoded form.
	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	err = pem.Encode(outFile, privateKey)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
func loadRSAPrivatePemKey(fileName string) *rsa.PrivateKey {
	privateKeyFile, err := os.Open(fileName)

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}

	pemfileinfo, _ := privateKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)
	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
	data, _ := pem.Decode([]byte(pembytes))
	privateKeyFile.Close()
	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
	return privateKeyImported
}
func loadPublicPemKey(fileName string) *rsa.PublicKey {

	publicKeyFile, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}

	pemfileinfo, _ := publicKeyFile.Stat()

	size := pemfileinfo.Size()
	pembytes := make([]byte, size)
	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pembytes)
	data, _ := pem.Decode([]byte(pembytes))
	publicKeyFile.Close()
	publicKeyFileImported, err := x509.ParsePKCS1PublicKey(data.Bytes)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
	return publicKeyFileImported
}
func generateRSAkeys(user string) {
	PrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println(err.Error)
		os.Exit(1)
	}
	PublicKey := PrivateKey.PublicKey
	way := "C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/keys/" + user
	if _, err := os.Stat(way + "/public_rsa.pem"); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(way+"/", 0777)
		_, err := os.Create(way + "/private_rsa.pem")
		_, err = os.Create(way + "/public_rsa.pem")
		if err != nil {
			panic(err)
		}
	}
	savePKCS8RSAPEMKey(way+"/private_rsa.pem", PrivateKey)
	savePublicPEMKey(way+"/public_rsa.pem", PublicKey)
}
func loadReceiverPublicRSAkey(encrypt_struct encrypt_struct, reciever string) encrypt_struct {
	importedRSAPublicKey := *loadPublicPemKey("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/keys/" + reciever + "/public_rsa.pem")
	encrypt_struct.receiverPublicKey_rsa = importedRSAPublicKey
	return encrypt_struct
}
func loadRecieverPrivateRSAkey(decrypt_struct decrypt_struct, reciever string) decrypt_struct {
	importedRSAPrivateKey := *loadRSAPrivatePemKey("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/keys/" + reciever + "/private_rsa.pem")
	decrypt_struct.PrivateKey_rsa = importedRSAPrivateKey
	return decrypt_struct
}
func loadSenderPrivateRSAkey(encrypt_struct encrypt_struct, sender string) encrypt_struct {
	importedRSAPrivateKey := *loadRSAPrivatePemKey("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/keys/" + sender + "/private_rsa.pem")
	encrypt_struct.senderPrivateKey_rsa = importedRSAPrivateKey
	return encrypt_struct
}
func loadSenderPublicRSAkey(decrypt_struct decrypt_struct, sender string) decrypt_struct {
	importedRSAPublicKey := *loadPublicPemKey("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/keys/" + sender + "/public_rsa.pem")
	decrypt_struct.senderPublicKey_rsa = importedRSAPublicKey
	return decrypt_struct
}

// RSA VERIFY
func SignPSS(encrypt_struct encrypt_struct, file []byte) []byte {
	// crypto/rand.Reader is a good source of entropy for blinding the RSA operation.
	rng := rand.Reader
	hashed := md5.Sum(file)
	var opts rsa.PSSOptions
	signature, err := rsa.SignPSS(rng, &encrypt_struct.senderPrivateKey_rsa, crypto.MD5, hashed[:], &opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from signing: %s\n", err)
	}
	return signature
}
func VerifyPSS(decrypt_struct decrypt_struct, signature []byte) bool {
	hashed := md5.Sum(decrypt_struct.body_decrypt)
	var opts rsa.PSSOptions
	err := rsa.VerifyPSS(&decrypt_struct.senderPublicKey_rsa, crypto.MD5, hashed[:], signature, &opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from verification: %s\n", err)
		return false
	}
	return true

}

func VerifyPSSAttach(decrypt_struct decrypt_struct, body []byte, signature []byte) bool {
	hashed := md5.Sum(body)
	var opts rsa.PSSOptions
	_ = rsa.VerifyPSS(&decrypt_struct.senderPublicKey_rsa, crypto.MD5, hashed[:], signature, &opts)
	return true

}

// OTHER
func GenerateRandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		index, _ := rand.Int(rand.Reader, big.NewInt(62))
		in1, _ := strconv.Atoi(index.String())
		s[i] = letters[in1]
	}
	return string(s)
}
func saveFile(source []byte, way string, file string) {

	if _, err := os.Stat("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + way); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/"+way, 0777)
	}
	f, err := os.Create("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + way + file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = f.Write(source)
	if err != nil {
		log.Fatal(err)
	}

}

func loadFile(way string) []byte {
	// Open file for reading
	f, err := os.Open("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + way)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	return data

}

//func menu() {
//for {
//var menu_key string
//println("[1] Зашифровать текст Боба и отправить Алисе")
//println("[2] Расшифровать текст,отправленный Бобом Алисе")

//fmt.Fscan(os.Stdin, &menu_key)
//switch menu_key {
//case "1":
//b, err := ioutil.ReadFile("input.txt")
//if err != nil {
//	fmt.Print(err)
//}
//	inputStr := string(b)

//var encrypt_struct encrypt_struct
//encrypt_struct = encryptdes(inputStr, encrypt_struct)
//encrypt_struct = loadSenderPrivateRSAkey(encrypt_struct)
//encrypt_struct = loadReceiverPublicRSAkey(encrypt_struct)
//encrypt_struct = EncryptOAEP(encrypt_struct)
//encrypt_struct = SignPSS(encrypt_struct)
//saveFile(encrypt_struct.des_encrypt_key[:], "key")
//saveFile(encrypt_struct.iv_des, "iv_des")
//saveFile(encrypt_struct.body_encrypt, "body_encrypt")
//saveFile(encrypt_struct.signature, "signature")
//case "2":
//decrypt rsa start
//var decrypt_struct decrypt_struct
//decrypt_struct.des_encrypt_key = loadFile("key")
//decrypt_struct.iv_des = loadFile("iv_des")
//decrypt_struct.cryptoText_des = loadFile("body_encrypt")
//decrypt_struct.signature = loadFile("signature")
//decrypt_struct = loadRecieverPrivateRSAkey(decrypt_struct)
//decrypt_struct = DecryptOAEP(decrypt_struct)
//decrypt_struct = decryptdes(decrypt_struct)
//fmt.Println("Расшифрованная фраза : " + string(decrypt_struct.body_decrypt))
//decrypt_struct = loadSenderPublicRSAkey(decrypt_struct)

//VerifyPSS(decrypt_struct)

//	}
//}
//}

[中文](README.md)
### Description
> It is crucial to securely store these sensitive information in our daily projects, 
> such as user passwords, various login authentication tokens, and so on. Ensuring 
> the secure storage of these sensitive information is of utmost importance.

### Methods
#### inner
> the `inner.py` script. The final solution provides an encryption function and 
> a decryption function, which allow users to perform encryption and decryption 
> operations by directly passing the values. 

> Before writing data, users can call the encryption function to encrypt the data. 
> After obtaining the encrypted data, users can store it according to their own scenarios.

#### Example
```python
    km = KeyManage()

    # init
    res1, ok1 = km.init()
    # Example
    t_plaintext = b"Hello, World!"  # The plaintext to encrypt
    c = InnerCrypt()
    t_ciphertext, s1 = c.encrypt(t_plaintext)
    print("Ciphertext:", t_ciphertext)
    decrypted_plaintext, s2 = c.decrypt(t_ciphertext)
    print(decrypted_plaintext)
    print(s2)
    print("Decrypted plaintext:", decrypted_plaintext)
    print(".......")
```

#### vault
> The prerequisite for using this feature is to deploy a third-party 
> password management software like Vault or have an existing Vault service.
> Sensitive information is stored directly in the third-party password 
> management software. 

> The current program only needs to configure the service address and token, 
> and then execute the methods for CRUD operations.

##### Prerequisites for usage
1. If you have started the Vault service, create a KV secret engine with a mount path of "cmdb" and start a Transit secret engine. Alternatively, you can choose to use the enable_secrets_engine command to automatically create them.
2. On the program side, configure the Vault address and provide the X-Vault-Token for API calls.

##### Example
```python
_base_url = "http://localhost:8200"
_token = "your token"
# Example
sdk = VaultClient(_base_url, _token, mount_path="cmdb")

_data = {"key1": "value1", "key2": "value2", "key3": "value3"}
_path = "test001"
_data = sdk.update(_path, _data, overwrite=True, encrypt=True)
_data = sdk.read(_path, decrypt=True)
```

### Other Use cases
> 如果您想在flask项目中使用它，可以参考开源项目[veops/cmdb](http://github.com/veops/cmdb)中
> 关于密码部分的使用，包括fask command 和api中对相应的功能的调用
[English](README_en.md)
### 描述
> 在我们的日常项目中，经常会遇得到一些敏感数据的存储问题，比如用户的密码，各种登录认证的token等等，
> 如何安全的存储这些敏感信息十分重要。

> 本工具是基于python实现的存储敏感数据方案，主要负责数据的加解密以及密钥维护等，
> 包括inner和vault两个方法，你可以在项目中直接使用这些代码来进行敏感信息的处理，
> 减少不必要的研究和开发。

### 两种方法
#### inner
> `inner.py` 脚本， 该方案最终使用中提供一个加密函数和一个解密函数函数，通过直接传递值进行加解密操作。
> 用户在进行写入数据之前通过调用加密函数对数据加密，获取到加密数据以后，用户再根据自己场景进行存储。

#### 使用示例
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
> 本功能使用前提是需要首先部署第三方密码管理软件vault或者已经有了vault的服务，
> 将敏感信息直接存储在第三方密码管理软件上，当前程序只需要配置服务地址以及令牌，
> 然后执行增删改查的方法即可。

##### 使用前提
1. 启动了vault服务，创建一个kv密码引擎，同时启动一个transit密码引擎，如创建一个mount_path为cmdb的kv,同事启动一个transit引擎；也可以选择使用执行enable_secrets_engine来自动创建。
2. 程序侧配置vault的地址，以及调用api的X-Vault-Token

##### 使用示例
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

### 其他使用场景
> 如果您想在flask项目中使用它，可以参考开源项目[veops/cmdb](http://github.com/veops/cmdb)中
> 关于密码部分的使用，包括flask command 和api中对相应的功能的调用

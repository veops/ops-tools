from base64 import b64encode, b64decode
from colorama import Back
from colorama import Fore
from colorama import Style
from colorama import init as colorama_init
from cryptography.hazmat.backends import default_backend
from cryptography.hazmat.primitives.ciphers import Cipher
from cryptography.hazmat.primitives.ciphers import algorithms
from cryptography.hazmat.primitives.ciphers import modes
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives import padding
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC

import os
import secrets
import sys
from Cryptodome.Protocol.SecretSharing import Shamir

# global_root_key just for test here
global_root_key = ""
global_encrypt_key = ""
global_iv_length = 16
global_key_shares = 5  # Number of generated key shares
global_key_threshold = 3  # Minimum number of shares required to rebuild the key
global_shares = []

backend_root_key_name = "root_key"
backend_encrypt_key_name = "encrypt_key"
backend_root_key_salt_name = "root_key_salt"
backend_encrypt_key_salt_name = "encrypt_key_salt"
success = "success"
seal_status = True
cache = {}


def string_to_bytes(value):
    if isinstance(value, bytes):
        return value
    if sys.version_info.major == 2:
        byte_string = value
    else:
        byte_string = value.encode("utf-8")
    return byte_string


class cache_backend:
    def __init__(self):
        pass

    @classmethod
    def get(cls, key):
        global cache
        return cache.get(key)

    @classmethod
    def add(cls, key, value):
        cache[key] = value
        return success, True

    @classmethod
    def update(cls, key, value):
        cache[key] = value
        return success, True


class Backend:
    def __init__(self, backend=None):
        if not backend:
            self.backend = cache_backend
        else:
            self.backend = backend

    def get(self, key):
        return self.backend.get(key)

    def add(self, key, value):
        return self.backend.add(key, value)


class KeyManage:

    def __init__(self, trigger=None):
        self.trigger = trigger
        pass

    @classmethod
    def hash_root_key(cls, value):
        algorithm = hashes.SHA256()
        salt = Backend().get(backend_root_key_salt_name)
        if not salt:
            salt = secrets.token_hex(16)
            msg, ok = Backend().add(backend_root_key_salt_name, salt)
            if not ok:
                return msg, ok
        kdf = PBKDF2HMAC(
            algorithm=algorithm,
            length=32,
            salt=string_to_bytes(salt),
            iterations=100000,
        )
        key = kdf.derive(string_to_bytes(value))
        return b64encode(key).decode('utf-8'), True

    @classmethod
    def generate_encrypt_key(cls, key):
        algorithm = hashes.SHA256()
        salt = Backend().get(backend_encrypt_key_salt_name)
        if not salt:
            salt = secrets.token_hex(32)
        kdf = PBKDF2HMAC(
            algorithm=algorithm,
            length=32,
            salt=string_to_bytes(salt),
            iterations=100000,
            backend=default_backend()
        )
        key = kdf.derive(string_to_bytes(key))
        msg, ok = Backend().add(backend_encrypt_key_salt_name, salt)
        if ok:
            return b64encode(key).decode('utf-8'), ok
        else:
            return msg, ok

    @classmethod
    def generate_keys(cls, secret):
        shares = Shamir.split(global_key_threshold, global_key_shares, secret)
        new_shares = []
        for share in shares:
            t = [i for i in share[1]] + [ord(i) for i in "{:0>2}".format(share[0])]
            new_shares.append(b64encode(bytes(t)))
        return new_shares

    def auth_root_secret(self, root_key):
        root_key_hash, ok = self.hash_root_key(b64encode(root_key))
        if not ok:
            return {
                "message": root_key_hash,
                "status": "failed"
            }
        backend_root_key_hash = Backend().get(backend_root_key_name)
        if not backend_root_key_hash:
            return {
                "message": "should init firstly",
                "status": "failed"
            }
        elif backend_root_key_hash != root_key_hash:
            return {
                "message": "invalid unseal keys",
                "status": "failed"
            }
        encrypt_key_aes = Backend().get(backend_encrypt_key_name)
        if not encrypt_key_aes:
            return {
                "message": "encrypt key is empty",
                "status": "failed"
            }
        global global_encrypt_key
        global global_root_key
        global global_shares
        global_encrypt_key = InnerCrypt.aes_decrypt(string_to_bytes(root_key), encrypt_key_aes)
        global_root_key = root_key
        global_shares = []
        return {
            "message": success,
            "status": success
        }

    def unseal(self, key):
        if not self.is_seal():
            return {
                "message": "current status is unseal, skip",
                "status": "skip"
            }
        global global_shares, global_root_key, global_encrypt_key
        t = [i for i in b64decode(key)]
        global_shares.append((int("".join([chr(i) for i in t[-2:]])), bytes(t[:-2])))
        if len(global_shares) >= global_key_threshold:
            recovered_secret = Shamir.combine(global_shares[:global_key_threshold])
            return self.auth_root_secret(recovered_secret)
        else:
            return {
                "Process": "{0}/{1}".format(len(global_shares), global_key_threshold),
                "message": "waiting for inputting other unseal key",
                "status": "waiting"
            }

    def generate_unseal_keys(self):
        info = Backend().get(backend_root_key_name)
        if info:
            return "already exist", [], False
        secret = AESGCM.generate_key(128)
        shares = self.generate_keys(secret)
        return b64encode(secret), shares, True

    def init(self):
        """
        init the master key, unseal key and store in backend
        :return:
        """
        root_key = Backend().get(backend_root_key_name)
        if root_key:
            return {"message": "already init, skip"}, False
        else:
            root_key, shares, status = self.generate_unseal_keys()
            if not status:
                return {"message": root_key}, False
            # hash root key and store in backend
            root_key_hash, ok = self.hash_root_key(root_key)
            if not ok:
                return {"message": root_key_hash}, False
            msg, ok = Backend().add(backend_root_key_name, root_key_hash)
            if not ok:
                return {"message": msg}, False
            # generate encrypt key from root_key and store in backend
            encrypt_key, ok = self.generate_encrypt_key(root_key)
            if not ok:
                return {"message": encrypt_key}
            encrypt_key_aes = InnerCrypt.aes_encrypt(root_key, encrypt_key)
            msg, ok = Backend().add(backend_encrypt_key_name, encrypt_key_aes)
            if not ok:
                return {"message": msg}, False
            #
            global global_root_key, global_encrypt_key
            global_root_key = root_key
            global_encrypt_key = encrypt_key
            self.print_token(shares, root_token=root_key)
            return {"message": "OK",
                    "details": {
                        "root_token": root_key,
                        "seal_tokens": shares,
                    }}, True

    def auto_unseal(self):
        if not self.trigger:
            return "trigger config is empty, skip", False
        if self.trigger.startswith("http"):
            pass
            #  TODO
        elif len(self.trigger.strip()) == 24:
            res = self.auth_root_secret(self.trigger)
            if res.get("status") == success:
                return success, True
            else:
                return res.get("message"), False
        else:
            return "trigger config is invalid, skip", False

    def seal(self, root_key):
        root_key_hash, ok = self.hash_root_key(b64encode(root_key))
        if not ok:
            return root_key_hash, ok
        backend_root_key_hash = Backend().get(backend_root_key_name)
        if not backend_root_key_hash:
            return "not init, seal skip", False
        else:
            global global_root_key
            global global_encrypt_key
            global_root_key = ""
            global_encrypt_key = ""
            return success, True

    @classmethod
    def is_seal(cls):
        """
        If there is no initialization or the root key is inconsistent, it is considered to be in a sealed state.
        :return:
        """
        root_key = Backend().get(backend_root_key_name)
        if root_key == "" or root_key != global_root_key:
            return "invalid root key", True
        return "", False

    @classmethod
    def print_token(cls, shares, root_token):
        """
        data: {"message": "OK",
               "details": {
                    "root_token": root_key,
                    "seal_tokens": shares,
              }}
        """
        colorama_init()
        print(Style.BRIGHT, "Please be sure to store the Unseal Key in a secure location and avoid losing it."
                            " The Unseal Key is required to unseal the system every time when it restarts."
                            " Successful unsealing is necessary to enable the password feature.")
        for i, v in enumerate(shares):
            print(
                Fore.RED + Back.LIGHTRED_EX + "unseal token " + str(i + 1) + ": " + v.decode("utf-8") + Style.RESET_ALL)
            print()
        print(Fore.GREEN + "root token:  " + root_token.decode("utf-8") + Style.RESET_ALL)


class InnerCrypt:
    def __init__(self, trigger=None):
        self.encrypt_key = b64decode(global_encrypt_key.encode("utf-8"))

    def encrypt(self, plaintext):
        """
        encrypt method contain aes currently
        """
        return self.aes_encrypt(self.encrypt_key, plaintext)

    def decrypt(self, ciphertext):
        """
        decrypt method contain aes currently
        """
        return self.aes_decrypt(self.encrypt_key, ciphertext)

    @classmethod
    def aes_encrypt(cls, key, plaintext):
        if isinstance(plaintext, str):
            plaintext = string_to_bytes(plaintext)
        iv = os.urandom(global_iv_length)
        try:
            cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
            encryptor = cipher.encryptor()
            v_padder = padding.PKCS7(algorithms.AES.block_size).padder()
            padded_plaintext = v_padder.update(plaintext) + v_padder.finalize()
            ciphertext = encryptor.update(padded_plaintext) + encryptor.finalize()
            return b64encode(iv + ciphertext).decode("utf-8"), True
        except Exception as e:
            return str(e), False

    @classmethod
    def aes_decrypt(cls, key, ciphertext):
        try:
            s = b64decode(ciphertext.encode("utf-8"))
            iv = s[:global_iv_length]
            ciphertext = s[global_iv_length:]
            cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
            decrypter = cipher.decryptor()
            decrypted_padded_plaintext = decrypter.update(ciphertext) + decrypter.finalize()
            unpadder = padding.PKCS7(algorithms.AES.block_size).unpadder()
            plaintext = unpadder.update(decrypted_padded_plaintext) + unpadder.finalize()
            return plaintext.decode('utf-8'), True
        except Exception as e:
            return str(e), False



if __name__ == "__main__":

    km = KeyManage()

    # init
    res1, ok1 = km.init()
    # Example
    t_plaintext = b"Hello, World!"  # The plaintext to encrypt
    c = InnerCrypt()
    t_ciphertext, s1 = c.encrypt(t_plaintext)
    print("-"*30)
    print("Ciphertext:", t_ciphertext)
    decrypted_plaintext, s2 = c.decrypt(t_ciphertext)
    print("Decrypted plaintext:", decrypted_plaintext)

# -*- coding:utf-8 -*-

from __future__ import unicode_literals
import importlib
import json
import os
import socket
import subprocess
import sys
import threading
import time

try:
    import configparser
    import ipaddress
    import psutil
    import tempfile
except:
    pass

global_scan_ip = "10.0.2.15"  # Identify on which device the subnet scanning is performed.
cidrs = ["192.168.20.8/30"]  # Subnet to be scanned.
global_ports_range = "3306-3310,3320"  # ports to be scanned. such as "3306-3310,3320"

paths = ["/etc/my.cnf", "/etc/mysql/my.cnf"]
threading_number = 10
pip_modules = ["configparser", "ipaddress", "psutil", "tempfile"]
pip_index_url = 'https://pypi.douban.com/simple'


class Module:

    def __init__(self, modules=None, index_url=None):
        self.modules = modules
        self.index_url = pip_index_url
        if not modules:
            self.modules = pip_modules
        if not index_url:
            self.index_url = pip_index_url

    def check_install_modules(self):
        for v in self.modules:
            self.install_missing_module(v)

    def install_missing_module(self, module_name):
        try:
            importlib.import_module(module_name)
        except ImportError:
            try:
                subprocess.check_output(["pip", "--version"])
            except subprocess.CalledProcessError:
                print("pips has not been installed, try to install...")
                subprocess.check_call([sys.executable, "-m", "ensurepip"])

            try:
                subprocess.check_call([sys.executable, "-m", "pip", "install", '--index-url',
                                       self.index_url, module_name])
                print("module '{}' has been installed successfully.".format(module_name))
            except Exception as e:
                print("Failed to install module '{}': {}".format(module_name, str(e)))


class Mysql:

    def __init__(self):
        pass

    @classmethod
    def parse_version(cls, sock):
        sock.sendall(b"\x00\x00\x00\x0a\x03\x00\x00\x00\x1b")
        data = sock.recv(1024)
        for encoding in ["latin-1", "utf-8", "ascii"]:
            try:
                d = data[5:].decode(encoding)
                if "MYSQL" in d.upper():
                    return data[5:].decode(encoding).split("\x00")[0]
                return ""
            except socket.error:
                continue
        return ""


class Scan:

    def __init__(self, cidrs, port_range, paths=None,  g_scan_ip=None):
        self.cidrs = cidrs
        self.ports = Utils.parse_port_range(port_range)
        self.local_ip = Utils.get_local_ip()
        if not paths:
            self.config_paths = ['/etc/mysql/my.cnf']
        else:
            self.config_paths = paths
        self.global_scan_ip = g_scan_ip

    def get_from_ip_range(self):
        instances = []
        if self.global_scan_ip != self.local_ip:
            return instances

        semaphore = threading.Semaphore(threading_number)

        def worker(scan_ip, s_port):
            r1 = self.scan_port(scan_ip, s_port)
            if r1:
                instances.append(r1)
            semaphore.release()

        threads = []
        for cidr in self.cidrs:
            ip_network = ipaddress.ip_network(cidr)

            for ip in ip_network.hosts():
                ip = str(ip)

                for port in self.ports:
                    semaphore.acquire()

                    t = threading.Thread(target=worker, args=(ip, port))
                    t.daemon = True
                    threads.append(t)
                    t.start()

        for i in threads:
            i.join()

        return instances

    @classmethod
    def scan_port(cls, ip, port):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(5)

            if sock.connect_ex((ip, port)) == 0:
                version = Mysql.parse_version(sock)
                if version:
                    return ip, port, version

            sock.close()
        except socket.error:
            return None
        return None

    @classmethod
    def local_listening_ports(cls):
        listening_ports = []
        for conn in psutil.net_connections():
            if conn.status == 'LISTEN':
                listening_ports.append(conn.laddr.port)
        return listening_ports

    def get_local_listening_ports(self):
        listening_ports = self.local_listening_ports()
        instances = []

        def worker(scan_ip, scan_port):
            r1 = self.scan_port(scan_ip, scan_port)
            if r1:
                instances.append(r1)
            semaphore.release()

        semaphore = threading.Semaphore(threading_number)
        threads = []
        for port in listening_ports:
            semaphore.acquire()

            t = threading.Thread(target=worker, args=(self.local_ip, port))
            t.daemon = True
            threads.append(t)
            t.start()

        for i in threads:
            i.join()

        return instances

    # this function will deprecate in the future
    def get_from_config(self):
        instances = []
        if self.local_ip == "":
            return instances
        for path in self.config_paths:
            config = configparser.ConfigParser()
            config.read(path)

            if 'mysqld' in config:
                port = config['mysqld'].get("port", "3306")
                pid_file = config['mysqld'].get("pid-file")

                pid = Utils.get_pid(pid_file)
                if pid:
                    ok, name = Utils.is_pid_running(pid)
                    if ok and name == "mysqld":
                        instances.append((self.local_ip, port, ""))

        return instances

    def gets(self):
        instances = self.get_from_ip_range()

        local_instance = self.get_local_listening_ports()
        if local_instance:
            instances.extend(local_instance)

        config_instances = self.get_from_config()
        if config_instances:
            instances.extend(config_instances)
        return Utils.deduplicate(instances)

class Cache:

    def __init__(self, temp_file=None, duration=None):
        if not temp_file:
            self.temp_file = os.path.join(tempfile.gettempdir(), "auto_discover_mysql_result.json")
        else:
            self.temp_file = temp_file
        if duration and isinstance(duration, int) and duration > 3600:
            self.duration = duration
        else:
            self.duration = 3600

    @classmethod
    def convert_data(cls, data):
        data = {
            "create_at": int(time.time()),
            "results": data,
        }
        return data

    def out_date(self, data):
        if isinstance(data, dict) and time.time() - data.get("create_at", 0) > self.duration:
            return True
        return False

    def read(self):
        try:
            with open(self.temp_file, 'r') as temp_file:
                json_data = json.load(temp_file)
        except:
            json_data = {}
        return json_data

    def write(self, data):
        with open(self.temp_file, "w") as temp_file:
            json.dump(data, temp_file)


class Utils:

    def __init__(self):
        pass

    @classmethod
    def parse_port_range(cls, port_range):
        ports = []
        ranges = port_range.split(",")
        for r in ranges:
            if "-" in r:
                start, end = r.split("-")
                try:
                    start = int(start)
                    end = int(end)
                    ports.extend(range(start, end + 1))
                except ValueError:
                    print("invalid port:", r)
            else:
                try:
                    port = int(r)
                    ports.append(port)
                except ValueError:
                    print("invalid port:", r)
        return ports

    @classmethod
    def get_local_ip(cls):
        interfaces = psutil.net_if_addrs()
        for interface in interfaces:
            addresses = interfaces[interface]
            for address in addresses:
                if address.family == socket.AF_INET and \
                        not address.address.startswith('127.') and \
                        not address.address.startswith('172.'):
                    return address.address
        return ""

    @classmethod
    def is_pid_running(cls, pid):
        if not pid:
            return False, ""
        try:
            process = psutil.Process(pid)
            return process.is_running(), process.name()
        except psutil.NoSuchProcess:
            return False, ""

    @classmethod
    def get_pid(cls, pid_file_path):
        try:
            with open(pid_file_path, "r") as file:
                pid = file.read().strip()
                return int(pid)
        except FileNotFoundError:
            return None
        except IOError as e:
            return None

    @classmethod
    def deduplicate(cls, data):
        unique_data = {}
        for item in data:
            key = (item[0], item[1])
            if item[2] != "":
                unique_data[key] = item
        return list(unique_data.values())

    @classmethod
    def convert_input(cls):
        if len(sys.argv) != 5:
            return

        c_cidrs = list(set(sys.argv[1].split(",")))

        c_ports = [3306]
        for item in sys.argv[2].split(","):
            try:
                c_ports.append(int(item))
            except ValueError:
                continue
        c_ports = list(set(c_ports))

        c_paths = list(set(sys.argv[3].split(",")))

        return c_cidrs, c_ports, c_paths, sys.argv[4]


class AutoDiscovery(object):

    @property
    def unique_key(self):
        return "mysql_name"

    @staticmethod
    def attributes():
        """
        :return: 返回属性字段列表, 列表项是(名称, 类型, 描述), 名称必须是英文
        类型: String Integer Float Date DateTime Time JSON
        """
        return [
            ("mysql_name", "String", "实例名称"),
            ("ip", "String", "ip"),
            ("port", "Integer", "端口"),
            ("version", "String", "版本信息")
        ]

    @staticmethod
    def run():
        """
        执行入口, 返回采集的属性值
        :return: 返回一个列表, 列表项是字典, 字典key是属性名称, value是属性值
        例如:
        return [dict(ci_type="server", private_ip="192.168.1.1")]
        """
        Module().check_install_modules()
        data = Cache().read()
        if not data or Cache().out_date(data):
            all_instances = Scan(cidrs, global_ports_range, paths, g_scan_ip=global_scan_ip).gets()
            results = []
            for instance in all_instances:
                ip, port, version = instance
                results.append(dict(mysql_name="{0}-{1}".format(ip, port), ip=ip, port=port, version=version))
            data = Cache().convert_data(results)
            Cache().write(data)
        return data.get("results", [])


if __name__ == "__main__":
    result = AutoDiscovery().run()
    if isinstance(result, list):
        print("AutoDiscovery::Result::{}".format(json.dumps(result)))
    else:
        print("ERROR: 采集返回必须是列表")

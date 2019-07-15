#include <iostream>
#include <string>
#include <sstream>
#include <stdexcept>
#include <winsock2.h>
#include <ws2tcpip.h>
#include <vector>

#include <openssl/ssl.h>
#include <openssl/err.h>

#define LOG std::cout

void check_error(const std::string &msg, bool failed) {
    if (failed) 
        throw std::runtime_error(msg);
}

void check_wsa_error(const std::string &msg, bool failed) {
    if (failed) {
        std::ostringstream out;
        out << "WSA " << msg << " failed with errno " << WSAGetLastError();
        check_error(out.str(), true);
    }
}

std::string ssl_err_as_string() {
    BIO *bio = BIO_new(BIO_s_mem());
    ERR_print_errors(bio);
    char *buf = NULL;
    size_t len = BIO_get_mem_data(bio, &buf);
    std::string ret(buf, len);
    BIO_free(bio);
    return ret;
}

void check_ssl_error(const std::string &msg, bool failed) {
    if (failed) {
        std::ostringstream out;
        out << "SSL " << msg << " failed with error: " << ssl_err_as_string();
        check_error(out.str(), true);
    }
}

void init_library() {
    WSADATA wsa;
    check_wsa_error("startup", WSAStartup(MAKEWORD(2, 2), &wsa) != 0);
    SSL_library_init();
    OpenSSL_add_all_algorithms();
    SSL_load_error_strings();
}

void cleanup_library() {
    check_wsa_error("cleanup", WSACleanup() == SOCKET_ERROR);
}


std::wstring Utf8ToUnicode(const std::string& utf8) {
    LPCCH ptr = utf8.c_str();
    int size = MultiByteToWideChar(CP_UTF8, 0, ptr, -1, NULL, NULL);
    std::wstring wstrRet(size, 0);
    int len = MultiByteToWideChar(CP_UTF8, 0, ptr, -1, (LPWSTR)wstrRet.c_str(), size);
    return wstrRet;
}

void showCerts(SSL * ssl) {
    char line[1024];
    X509 *cert = SSL_get_peer_certificate(ssl);
    if (cert != NULL) {
        LOG << "have X.509 certificate!" << std::endl;
        std::memset(line, 0, sizeof(line));
        X509_NAME_oneline(X509_get_subject_name(cert), line, sizeof(line) - 1);
        LOG << "  [SUBJECT] " << line << std::endl;
        std::memset(line, 0, sizeof(line));
        X509_NAME_oneline(X509_get_issuer_name(cert), line, sizeof(line) - 1);
        LOG << "  [ISSUER ] " << line << std::endl;
        X509_free(cert);
    }
    else
        LOG << "have no X.509 certificate!" << std::endl;
}

class MailerConn {
public:
    MailerConn(const std::string &_host, const std::string &_port, bool secure)
            : host(_host), port(_port), useTLS(secure) {

        addrinfo hints;
        std::memset(&hints, 0, sizeof(addrinfo));
        hints.ai_family = AF_INET;
        hints.ai_socktype = SOCK_STREAM;
        hints.ai_flags = 0;
        hints.ai_protocol = IPPROTO_TCP;
        addrinfo *result;
        check_wsa_error("getaddrinfo", getaddrinfo(host.c_str(), port.c_str(), &hints, &result) != 0);
        for (addrinfo *rp = result; rp != NULL; rp = rp->ai_next) {
            // LOG << "port = " << ntohs(((sockaddr_in*)rp->ai_addr)->sin_port) << std::endl;
            sock = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
            if (sock == INVALID_SOCKET)
                break;
            if (connect(sock, rp->ai_addr, (int)rp->ai_addrlen) == SOCKET_ERROR) {
                closesocket(sock);
                sock = INVALID_SOCKET;
                continue;
            }
            break;
        }
        freeaddrinfo(result);
        check_error(std::string("connect to ")+host+":"+port+" failed", sock == SOCKET_ERROR);

        if (useTLS) 
            setup_TLS();
    }

    ~MailerConn() {
        if (useTLS) {
            SSL_shutdown(ssl);
            SSL_free(ssl);
            SSL_CTX_free(ctx);
        }
        closesocket(sock);
    }

    void setup_TLS() {
        ctx = SSL_CTX_new(SSLv23_client_method());
        check_ssl_error("context", ctx == NULL);
        ssl = SSL_new(ctx);
        SSL_set_fd(ssl, (int)sock);
        check_ssl_error("setfd", SSL_connect(ssl) == -1);
    }

    void turn_to_TLS() {
        if (!useTLS) {
            useTLS = true;
            setup_TLS();
        }
    }

    int read(char *buf, int num) {
        int len;
        if (useTLS) {
            len = SSL_read(ssl, buf, num);
            check_ssl_error("read", len < 0);
            
        }
        else {
            len = recv(sock, buf, num, 0);
            check_wsa_error("read", len < 0);
        }
        return len;
    }

    int write(const char *buf, int num) {
        int len;
        if (useTLS) {
            len = SSL_write(ssl, buf, num);
            check_ssl_error("write", len <= 0);

        }
        else {
            len = send(sock, buf, num, 0);
            check_wsa_error("write", len <= 0);
        }
        return len;
    }

    std::string get(const std::string &endFlag = "\r\n") {
        char buf[1024];
        int len;
        std::ostringstream ret;
        do {
            std::memset(buf, 0, sizeof(buf));
            len = read(buf, sizeof(buf)-1);
            ret << std::string(buf, len);
        } while (len != 0 && ret.str().find(endFlag) == std::string::npos);
        std::string retstr = ret.str();
        LOG << "[SERVER] " << retstr;
        return retstr;
    }

    std::string put(const std::string &content) {
        LOG << "[CLIENT] " << content;
        write(content.c_str(), (int)content.size());
        return get();
    }
   
private:
    std::string host, port;
    bool useTLS;
    SOCKET sock = INVALID_SOCKET;
    SSL_CTX *ctx = NULL;
    SSL *ssl = NULL;
};


std::string encode_base64(const std::string &s) {
    BIO *bmem, *b64;
    BUF_MEM *bptr;
    b64 = BIO_new(BIO_f_base64());
    bmem = BIO_new(BIO_s_mem());
    b64 = BIO_push(b64, bmem);
    BIO_write(b64, s.c_str(), (int)s.size());
    BIO_flush(b64);
    BIO_get_mem_ptr(b64, &bptr);
    std::string ret(bptr->data, bptr->length);
    BIO_free_all(b64);
    ret.pop_back();
    return ret;
}

std::string decode_base64(const std::string &_s) {
    std::string s = _s + "\n";
    int len = 0;
    BIO *b64, *bmem;
    b64 = BIO_new(BIO_f_base64());
    bmem = BIO_new_mem_buf(s.c_str(), (int)s.size());
    bmem = BIO_push(b64, bmem);
    char *buf = new char[s.size()]{0};
    len = BIO_read(bmem, buf, (int)s.size()-1);
    BIO_free_all(bmem);
    std::string ret(buf, len);
    delete[] buf;
    return ret;
}

class SMTPMail {
public:
    SMTPMail(const std::string &_host, const std::string &_port, bool _secure)
        : host(_host), port(_port), secure(_secure) {}

    void login(const std::string &_user, const std::string &_pass) {
        user = _user;
        authuser = encode_base64(_user);
        authpass = encode_base64(_pass);
    }

    void to(const std::string &peer) {
        recipients.push_back(peer);
    }

    void make(const std::string &subject, const std::string &content) {
        std::ostringstream mime;
        mime << "From: " << user << "\r\nTo: ";
        for (const auto &peer : recipients)
            mime << peer << ";";
        mime << "\r\nSubject: " << subject << "\r\n\r\n";
        mime << content;
        raw.assign(mime.str());
    }

    bool send(std::string &failed_reason) {
        try {
            MailerConn conn(host, port, secure);
            std::string resp;
            conn.get();
            resp = conn.put(std::string("EHLO ") + host + "\r\n");
            if (!secure && resp.find("STARTTLS") != std::string::npos) {
                conn.put("STARTTLS\r\n");
                conn.turn_to_TLS();
                conn.put(std::string("EHLO ") + host + "\r\n");
            }
            conn.put("AUTH LOGIN\r\n");
            conn.put(authuser + "\r\n");
            resp = conn.put(authpass + "\r\n");
            if (resp.find("successful") == std::string::npos) {
                failed_reason.assign(resp);
                return false;
            }
            conn.put(std::string("MAIL FROM: <") + user + ">\r\n");
            for (const auto &peer : recipients)
                conn.put("RCPT TO: <" + peer + ">\r\n");
            conn.put("DATA\r\n");
            resp = conn.put(raw + "\r\n.\r\n");
            if (resp.substr(0, 3) != "250") {
                failed_reason.assign(resp);
                return false;
            }
            conn.put("QUIT\r\n");
        }
        catch (std::runtime_error e) {
            failed_reason.assign(e.what());
            return false;
        }
        return true;
    }

private:
    bool secure;
    std::string host;
    std::string port;
    std::string user;
    std::string authuser;
    std::string authpass;
    std::vector<std::string> recipients;
    std::string raw;
};


int main() {
    //std::wcout.imbue(std::locale(""));
    init_library();

    SMTPMail m("smtp.qq.com", "465", true);
    m.login("jaysinco@qq.com", "oolmgpqhbvqyicfb");
    mail.to("jaysinco@163.com");
    mail.make("please report to me!", "we need money!");
    std::string failed_reason;
    bool ok = mail.send(failed_reason);
    std::cout << "===== mail send status: " << ok << " " << failed_reason << " ======\n";

    cleanup_library();
}



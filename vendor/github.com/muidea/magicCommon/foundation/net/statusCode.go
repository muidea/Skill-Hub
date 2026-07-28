package net

// http://www.httpstatus.cn/
// http://www.restapitutorial.com/httpstatuscodes.html

// Status_100_Continue 1×× Informational
// 100 Continue
// 客户端应当继续发送请求。这个临时响应是用来通知客户端它的部分请求已经被服务器接收，且仍未被拒绝。客户端应当继续发送请求的剩余部分，或者如果请求已经完成，忽略这个响应。服务器必须在请求完成后向客户端发送一个最终响应。
const Status_100_Continue = 100

// Status_101_SwitchProtocols 101 Switching Protocols
// 服务器已经理解了客户端的请求，并将通过Upgrade 消息头通知客户端采用不同的协议来完成这个请求。在发送完这个响应最后的空行后，服务器将会切换到在Upgrade 消息头中定义的那些协议。
// 只有在切换新的协议更有好处的时候才应该采取类似措施。例如，切换到新的HTTP 版本比旧版本更有优势，或者切换到一个实时且同步的协议以传送利用此类特性的资源。
const Status_101_SwitchProtocols = 101

// Status_102_Processing 102 Processing
// 由WebDAV（RFC 2518）扩展的状态码，代表处理将被继续执行。
const Status_102_Processing = 102

// Status_200_OK 2×× Success
// 200 OK
// 请求已成功，请求所希望的响应头或数据体将随此响应返回。
const Status_200_OK = 200

// Status_201_Created 201 Created
// 请求已经被实现，而且有一个新的资源已经依据请求的需要而建立，且其 URI 已经随Location 头信息返回。假如需要的资源无法及时建立的话，应当返回 '202 Accepted'。
const Status_201_Created = 201

// Status_202_Accepted 202 Accepted
// 服务器已接受请求，但尚未处理。正如它可能被拒绝一样，最终该请求可能会也可能不会被执行。在异步操作的场合下，没有比发送这个状态码更方便的做法了。
// 返回202状态码的响应的目的是允许服务器接受其他过程的请求（例如某个每天只执行一次的基于批处理的操作），而不必让客户端一直保持与服务器的连接直到批处理操作全部完成。
// 在接受请求处理并返回202状态码的响应应当在返回的实体中包含一些指示处理当前状态的信息，以及指向处理状态监视器或状态预测的指针，以便用户能够估计操作是否已经完成
const Status_202_Accepted = 202

// Status_203_NoneAuthoritative 203 Non-authoritative Information
// 服务器已成功处理了请求，但返回的实体头部元信息不是在原始服务器上有效的确定集合，而是来自本地或者第三方的拷贝。
// 当前的信息可能是原始版本的子集或者超集。例如，包含资源的元数据可能导致原始服务器知道元信息的超集。使用此状态码不是必须的，而且只有在响应不使用此状态码便会返回200 OK的情况下才是合适的。
const Status_203_NoneAuthoritative = 203

// Status_204_NoContent 204 No Content
// 服务器成功处理了请求，但不需要返回任何实体内容，并且希望返回更新了的元信息。响应可能通过实体头部的形式，返回新的或更新后的元信息。如果存在这些头部信息，则应当与所请求的变量相呼应。
// 如果客户端是浏览器的话，那么用户浏览器应保留发送了该请求的页面，而不产生任何文档视图上的变化，即使按照规范新的或更新后的元信息应当被应用到用户浏览器活动视图中的文档。
// 由于204响应被禁止包含任何消息体，因此它始终以消息头后的第一个空行结尾。
const Status_204_NoContent = 204

// Status_204_ResetContent 205 Reset Content
// 服务器成功处理了请求，且没有返回任何内容。但是与204响应不同，返回此状态码的响应要求请求者重置文档视图。该响应主要是被用于接受用户输入后，立即重置表单，以便用户能够轻松地开始另一次输入。
// 与204响应一样，该响应也被禁止包含任何消息体，且以消息头后的第一个空行结束。
const Status_204_ResetContent = 205

// Status_206_PartialContent 206 Partial Content
// 服务器已经成功处理了部分 GET 请求。类似于 FlashGet 或者迅雷这类的 HTTP 下载工具都是使用此类响应实现断点续传或者将一个大文档分解为多个下载段同时下载。
// 该请求必须包含 Range 头信息来指示客户端希望得到的内容范围，并且可能包含 If-Range 来作为请求条件。
// 响应必须包含如下的头部域：
// Content-Range 用以指示本次响应中返回的内容的范围；如果是 Content-Type 为 multipart/byteranges 的多段下载，则每一 multipart 段中都应包含 Content-Range 域用以指示本段的内容范围。假如响应中包含 Content-Length，那么它的数值必须匹配它返回的内容范围的真实字节数。
// Date ETag 和/或 Content-Location，假如同样的请求本应该返回200响应。
// Expires, Cache-Control，和/或 Vary，假如其值可能与之前相同变量的其他响应对应的值不同的话。
// 假如本响应请求使用了 If-Range 强缓存验证，那么本次响应不应该包含其他实体头；假如本响应的请求使用了 If-Range 弱缓存验证，那么本次响应禁止包含其他实体头；这避免了缓存的实体内容和更新了的实体头信息之间的不一致。否则，本响应就应当包含所有本应该返回200响应中应当返回的所有实体头部域。
// 假如 ETag 或 Last-Modified 头部不能精确匹配的话，则客户端缓存应禁止将206响应返回的内容与之前任何缓存过的内容组合在一起。
// 任何不支持 Range 以及 Content-Range 头的缓存都禁止缓存206响应返回的内容。
const Status_206_PartialContent = 206

// Status_207_MultiStatus 207 Multi-Status
// 由WebDAV(RFC 2518)扩展的状态码，代表之后的消息体将是一个XML消息，并且可能依照之前子请求数量的不同，包含一系列独立的响应代码。
const Status_207_MultiStatus = 207

// Status_208_AlreadyReported 208 Already Reported
// DAV绑定的成员已经在（多状态）响应之前的部分被列举，且未被再次包含。
const Status_208_AlreadyReported = 208

// Status_226_IMUsed 226 IM Used
// 服务器已经满足了对资源的请求，对实体请求的一个或多个实体操作的结果表示。
const Status_226_IMUsed = 226

// Status_300_MultipleChoices 3×× Redirection
// 300 Multiple Choices
// 被请求的资源有一系列可供选择的回馈信息，每个都有自己特定的地址和浏览器驱动的商议信息。用户或浏览器能够自行选择一个首选的地址进行重定向。
// 除非这是一个 HEAD 请求，否则该响应应当包括一个资源特性及地址的列表的实体，以便用户或浏览器从中选择最合适的重定向地址。这个实体的格式由 Content-Type 定义的格式所决定。浏览器可能根据响应的格式以及浏览器自身能力，自动作出最合适的选择。当然，RFC 2616规范并没有规定这样的自动选择该如何进行。
// 如果服务器本身已经有了首选的回馈选择，那么在 Location 中应当指明这个回馈的 URI；浏览器可能会将这个 Location 值作为自动重定向的地址。此外，除非额外指定，否则这个响应也是可缓存的。
const Status_300_MultipleChoices = 300

// Status_301_MovedPermanetly 301 Moved Permanently
// 被请求的资源已永久移动到新位置，并且将来任何对此资源的引用都应该使用本响应返回的若干个 URI 之一。如果可能，拥有链接编辑功能的客户端应当自动把请求的地址修改为从服务器反馈回来的地址。除非额外指定，否则这个响应也是可缓存的。
// 新的永久性的 URI 应当在响应的 Location 域中返回。除非这是一个 HEAD 请求，否则响应的实体中应当包含指向新的 URI 的超链接及简短说明。
// 如果这不是一个 GET 或者 HEAD 请求，因此浏览器禁止自动进行重定向，除非得到用户的确认，因为请求的条件可能因此发生变化。
// 注意：对于某些使用 HTTP/1.0 协议的浏览器，当它们发送的 POST 请求得到了一个301响应的话，接下来的重定向请求将会变成 GET 方式。
const Status_301_MovedPermanetly = 301

// Status_302_Found 302 Found
// 请求的资源现在临时从不同的 URI 响应请求。由于这样的重定向是临时的，客户端应当继续向原有地址发送以后的请求。只有在Cache-Control或Expires中进行了指定的情况下，这个响应才是可缓存的。
// 新的临时性的 URI 应当在响应的 Location 域中返回。除非这是一个 HEAD 请求，否则响应的实体中应当包含指向新的 URI 的超链接及简短说明。
// 如果这不是一个 GET 或者 HEAD 请求，那么浏览器禁止自动进行重定向，除非得到用户的确认，因为请求的条件可能因此发生变化。
// 注意：虽然RFC 1945和RFC 2068规范不允许客户端在重定向时改变请求的方法，但是很多现存的浏览器将302响应视作为303响应，并且使用 GET 方式访问在 Location 中规定的 URI，而无视原先请求的方法。状态码303和307被添加了进来，用以明确服务器期待客户端进行何种反应。
const Status_302_Found = 302

// Status_303_SessOther 303 See Other
// 对应当前请求的响应可以在另一个 URI 上被找到，而且客户端应当采用 GET 的方式访问那个资源。这个方法的存在主要是为了允许由脚本激活的POST请求输出重定向到一个新的资源。这个新的 URI 不是原始资源的替代引用。同时，303响应禁止被缓存。当然，第二个请求（重定向）可能被缓存。 　　新的 URI 应当在响应的 Location 域中返回。除非这是一个 HEAD 请求，否则响应的实体中应当包含指向新的 URI 的超链接及简短说明。
// 注意：许多 HTTP/1.1 版以前的 浏览器不能正确理解303状态。如果需要考虑与这些浏览器之间的互动，302状态码应该可以胜任，因为大多数的浏览器处理302响应时的方式恰恰就是上述规范要求客户端处理303响应时应当做的。
const Status_303_SessOther = 303

// Status_304_NotModified 304 Not Modified
// 如果客户端发送了一个带条件的 GET 请求且该请求已被允许，而文档的内容（自上次访问以来或者根据请求的条件）并没有改变，则服务器应当返回这个状态码
const Status_304_NotModified = 304

// Status_305_UseProxy 305 Use Proxy
// 被请求的资源必须通过指定的代理才能被访问
const Status_305_UseProxy = 305

// Status_307_TemporaryRedirect 307 Temporary Redirect
// 请求的资源现在临时从不同的URI 响应请求。由于这样的重定向是临时的，客户端应当继续向原有地址发送以后的请求
const Status_307_TemporaryRedirect = 307

// Status_308_PermanentRedirect 308 Permanent Redirect
// 请求和所有将来的请求应该使用另一个URI重复
const Status_308_PermanentRedirect = 308

// Status_400_BadRequest 4×× Client Error
// 400 Bad Request
// 1、语义有误，当前请求无法被服务器理解。除非进行修改，否则客户端不应该重复提交这个请求。
// 2、请求参数有误。
const Status_400_BadRequest = 400

// Status_401_Unauthorized 401 Unauthorized
// 当前请求需要用户验证
const Status_401_Unauthorized = 401

// Status_402_Payment_Required 402 Payment Required
// 该状态码是为了将来可能的需求而预留的
const Status_402_Payment_Required = 402

// Status_403_Forbidden 403 Forbidden
// 服务器已经理解请求，但是拒绝执行它
const Status_403_Forbidden = 403

// Status_404_NotFound 404 Not Found
// 请求失败，请求所希望得到的资源未被在服务器上发现
const Status_404_NotFound = 404

// Status_405_MethodNotAllowed 405 Method Not Allowed
// 请求行中指定的请求方法不能被用于请求相应的资源
const Status_405_MethodNotAllowed = 405

// Status_406_NotAcceptable 406 Not Acceptable
// 请求的资源的内容特性无法满足请求头中的条件，因而无法生成响应实体
const Status_406_NotAcceptable = 406

// Status_407_ProxyAuthenticationRequired 407 Proxy Authentication Required
// 与401响应类似，只不过客户端必须在代理服务器上进行身份验证
const Status_407_ProxyAuthenticationRequired = 407

// Status_408_RequestTimeout 408 Request Timeout
// 请求超时
const Status_408_RequestTimeout = 408

// Status_409_Conflict 409 Conflict
// 由于和被请求的资源的当前状态之间存在冲突，请求无法完成
const Status_409_Conflict = 409

// Status_410_Gone 410 Gone
// 被请求的资源在服务器上已经不再可用，而且没有任何已知的转发地址
const Status_410_Gone = 410

// Status_411_LengthRequired 411 Length Required
// 服务器拒绝在没有定义 Content-Length 头的情况下接受请求
const Status_411_LengthRequired = 411

// Status_412_ProconditionFailed 412 Precondition Failed
// 服务器在验证在请求的头字段中给出先决条件时，没能满足其中的一个或多个
const Status_412_ProconditionFailed = 412

// Status_413_PayloadTooLarge 413 Payload Too Large
// 服务器拒绝处理当前请求，因为该请求提交的实体数据大小超过了服务器愿意或者能够处理的范围
const Status_413_PayloadTooLarge = 413

// Status_414_RequestURITooLong 414 Request-URI Too Long
// 请求的URI 长度超过了服务器能够解释的长度，因此服务器拒绝对该请求提供服务
const Status_414_RequestURITooLong = 414

// Status_415_UnsupportedMediaType 415 Unsupported Media Type
// 对于当前请求的方法和所请求的资源，请求中提交的实体并不是服务器中所支持的格式，因此请求被拒绝
const Status_415_UnsupportedMediaType = 415

// Status_416_RequestedRangeNotSatisfiable 416 Requested Range Not Satisfiable
// 如果请求中包含了 Range 请求头，并且 Range 中指定的任何数据范围都与当前资源的可用范围不重合，同时请求中又没有定义 If-Range 请求头，那么服务器就应当返回416状态码
const Status_416_RequestedRangeNotSatisfiable = 416

// Status_417_ExpectationFailed 417 Expectation Failed
// 在请求头 Expect 中指定的预期内容无法被服务器满足，或者这个服务器是一个代理服务器，它有明显的证据证明在当前路由的下一个节点上，Expect 的内容无法被满足。
const Status_417_ExpectationFailed = 417

// Status_418_ImATeapot 418 I'm a teapot
// 本操作码是在1998年作为IETF的传统愚人节笑话, 在RFC 2324超文本咖啡壶控制协议'中定义的，并不需要在真实的HTTP服务器中定义。
// 當一個控制茶壺的HTCPCP收到BREW或POST指令要求其煮咖啡時應當回傳此錯誤。[49]这个HTTP状态码在某些网站（包括Google.com）與項目（如Node.js、ASP.NET和Go語言）中用作彩蛋。[50]
const Status_418_ImATeapot = 418

// Status_421_MisdirectedRequest 421 Misdirected Request
// 该请求针对的是无法产生响应的服务器（例如因为连接重用）
const Status_421_MisdirectedRequest = 421

// Status_422_UnprocessableEntity 422 Unprocessable Entity
// 请求格式正确，但是由于含有语义错误，无法响应。
const Status_422_UnprocessableEntity = 422

// Status_423_Locked 423 Locked
// 当前资源被锁定
const Status_423_Locked = 423

// Status_424_FailedDependency 424 Failed Dependency
// 由于之前的某个请求发生的错误，导致当前请求失败，例如PROPPATCH
const Status_424_FailedDependency = 424

// Status_426_UpgradeRequired 426 Upgrade Required
// 客户端应当切换到TLS/1.0
const Status_426_UpgradeRequired = 426

// Status_428_PreconditionRequired 428 Precondition Required
// 原服务器要求该请求满足一定条件
const Status_428_PreconditionRequired = 428

// Status_429_TooManyRequest 429 Too Many Requests
// 用户在给定的时间内发送了太多的请求。
const Status_429_TooManyRequest = 429

// Status_431_RquestHeaderFieldsTooLarge 431 Request Headers Fields Too Large
// 服务器不愿处理请求，因为一个或多个头字段过大
const Status_431_RquestHeaderFieldsTooLarge = 431

// Status_444_ConnectionClosedWithoutResponse 444 Connection Closed Without Response
// Nginx上HTTP服務器擴展。服務器不向客戶端返回任何信息
const Status_444_ConnectionClosedWithoutResponse = 444

// Status_451_UnavailableForLegalReason 451 Unavailable For Legal Reasons
// 该访问因法律的要求而被拒絕，由IETF在2015核准后新增加
const Status_451_UnavailableForLegalReason = 451

// Status_500_InternalServerError 5×× Server Error
// 500 Internal Server Error
// 通用错误消息，服务器遇到了一个未曾预料的状况，导致了它无法完成对请求的处理
const Status_500_InternalServerError = 500

// Status_501_NotImplemented 501 Not Implemented
// 服务器不支持当前请求所需要的某个功能
const Status_501_NotImplemented = 501

// Status_502_BadGateway 502 Bad Gateway
// 作为网关或者代理工作的服务器尝试执行请求时，从上游服务器接收到无效的响应
const Status_502_BadGateway = 502

// Status_503_ServiceUnavailable 503 Service Unavailable
// 由于临时的服务器维护或者过载，服务器当前无法处理请求
const Status_503_ServiceUnavailable = 503

// Status_504_GatewayTimeout 504 Gateway Timeout
// 作为网关或者代理工作的服务器尝试执行请求时，未能及时从上游服务器（URI标识出的服务器，例如HTTP、FTP、LDAP）或者辅助服务器（例如DNS）收到响应
const Status_504_GatewayTimeout = 504

// Status_505_HttpVersionNotSupported 505 HTTP Version Not Supported
// 服务器不支持，或者拒绝支持在请求中使用的HTTP版本
const Status_505_HttpVersionNotSupported = 505

// Status_506_VariantAlsoNegotiates 506 Variant Also Negotiates
// 由《透明内容协商协议》（RFC 2295）扩展，代表服务器存在内部配置错误：被请求的协商变元资源被配置为在透明内容协商中使用自己，因此在一个协商处理中不是一个合适的重点。
const Status_506_VariantAlsoNegotiates = 506

// Status_507_InsufficientStorage 507 Insufficient Storage
// 服务器无法存储完成请求所必须的内容
const Status_507_InsufficientStorage = 507

// Status_508_LoopDetected 508 Loop Detected
// 服务器在处理请求时陷入死循环
const Status_508_LoopDetected = 508

// Status_510_NotExtended 510 Not Extended
// 获取资源所需要的策略并没有被满足
const Status_510_NotExtended = 510

// Status_510_NetworkAuthenticationRequired 511 Network Authentication Required
// 客户端需要进行身份验证才能获得网络访问权限，旨在限制用户群访问特定网络
const Status_510_NetworkAuthenticationRequired = 511

// Status_599_NetworkConnectTimeout 599 Network Connect Timeout Error
const Status_599_NetworkConnectTimeout = 599

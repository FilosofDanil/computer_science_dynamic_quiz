# CS Foundations Pyramid — Software Engineer Guide (2026)

> A bottom-up learning path that demystifies the full stack. Each layer builds mental models that make every tool above it intuitive rather than magical.

---

## How to use this guide

Work through the layers in order. Each topic includes a one-sentence overview — your job is to get that intuition first, then go deeper on whatever your work demands. You don't need to master everything; you need to stop being surprised.

---

## Layer 1 — CPU & Hardware Primitives

*What the machine actually does, nothing more.*

### Topics

| Topic | Overview |
|---|---|
| ARM instruction set | A CPU can only fetch, store, add, compare, branch, and shift — everything else is combinations of these. |
| Registers vs memory | Registers are the CPU's handful of scratch variables; RAM is millions of times slower to access. |
| CPU cache hierarchy (L1/L2/L3) | Data already in L1 cache is ~100× faster to access than data in RAM — locality of reference is the single biggest performance lever. |
| Cache lines & false sharing | The CPU loads memory in fixed 64-byte chunks, so nearby data comes for free and unrelated data sharing a chunk causes invisible slowdowns. |
| Memory alignment | Misaligned data reads can require two cache-line fetches instead of one, silently halving throughput. |
| Branch prediction | The CPU speculatively executes the next instruction before a condition resolves — unpredictable branches flush the pipeline and cost dozens of cycles. |
| SIMD / vector instructions | A single instruction can operate on 4–16 values in parallel when data is laid out contiguously, which is why NumPy beats a Python loop by 100×. |

---

## Layer 2 — Memory Model & Concurrency Primitives

*How multiple things share one machine without destroying each other.*

### Topics

| Topic | Overview |
|---|---|
| Stack vs heap | The stack is automatic, fast, and scoped to a function call; the heap is manual (or GC-managed), slower, and lives as long as you keep a reference. |
| Pointers & references | A pointer is just a number — an address in virtual memory — and every "object reference" in any language is ultimately this. |
| Memory ordering & reordering | CPUs and compilers reorder instructions for speed, so without explicit barriers, two threads can observe writes in different orders. |
| Atomic operations | A single indivisible read-modify-write is the primitive that makes lock-free data structures possible. |
| Mutexes & condition variables | A mutex serialises access to shared state; a condition variable lets a thread sleep until that state changes. |
| Memory safety: dangling pointers, use-after-free | Accessing memory after it has been freed is undefined behaviour — the source of most serious security vulnerabilities in C/C++ code. |

---

## Layer 3 — C & Systems Programming

*The minimal language that maps 1:1 onto hardware concepts.*

### Topics

| Topic | Overview |
|---|---|
| Compilation pipeline (preprocessor → AST → IR → machine code) | Source code passes through four distinct transformations before becoming the bytes that the CPU actually executes. |
| Linking: static vs dynamic | Static linking bakes library code into your binary; dynamic linking loads it at runtime and lets multiple processes share one copy in memory. |
| File I/O and buffering | Reading a file in one large buffered call is orders of magnitude faster than calling `read()` one byte at a time, because each syscall is expensive. |
| `errno`, syscalls, and the kernel boundary | Every interaction with hardware or the OS goes through a system call — a controlled jump into kernel mode with a measurable overhead. |
| `struct` layout and padding | The compiler inserts invisible padding between fields to satisfy alignment requirements, so struct size is not just the sum of its members. |
| Manual memory management (`malloc`/`free`) | Understanding explicit allocation makes every GC language's behaviour and pauses intuitive. |

---

## Layer 4 — Runtimes, Compilers & Language Models

*Why Python is slow, why Go is fast, and what's actually happening in between.*

### Topics

| Topic | Overview |
|---|---|
| Compiled without runtime (C, Rust) | The binary runs directly on the CPU with no intermediary — what you wrote is what executes. |
| Compiled with runtime (Go, Java, C#) | A runtime ships alongside your code to manage goroutines/threads, GC, and reflection — fast, but with GC pauses and a startup cost. |
| Interpreted (Python, Ruby) | Each source line is decoded at runtime by an interpreter written in C — convenient but 10–100× slower for CPU-bound work. |
| JIT compilation (V8, JVM HotSpot, PyPy) | A just-in-time compiler profiles hot code paths at runtime and compiles them to native instructions, narrowing the gap with static compilers. |
| Garbage collection algorithms | GC strategies (mark-and-sweep, generational, reference counting) make different trade-offs between throughput, latency, and memory overhead. |
| Green threads vs OS threads | OS threads are scheduled by the kernel and expensive to create; green threads (goroutines, async tasks) are scheduled by the runtime and cheap enough to have millions of. |
| Async / event loops (Node.js, asyncio) | A single OS thread can handle thousands of concurrent I/O operations by registering callbacks and never blocking — the tradeoff is that CPU-bound work blocks everyone. |

---

## Layer 5 — Operating Systems & the Kernel

*The software that makes one physical machine look like many isolated ones.*

### Topics

| Topic | Overview |
|---|---|
| Kernel vs userspace | The kernel has unrestricted hardware access; user processes live in a protected sandbox and must ask the kernel for resources via syscalls. |
| Processes & process isolation | Each process has its own virtual address space — one process cannot read another's memory without the kernel's explicit permission. |
| Virtual memory & paging | The OS maps each process's logical addresses to physical RAM pages, allowing programs larger than RAM and making address-space layout randomisation (ASLR) possible. |
| Scheduling (preemptive, CFS) | The kernel's scheduler decides which process runs on which CPU core at any microsecond — fairness and latency are competing goals. |
| File system abstractions (VFS, inodes) | "Everything is a file" works because the kernel presents sockets, pipes, and devices through the same read/write interface as disk files. |
| Signals and inter-process communication (pipes, sockets, shared memory) | Processes communicate via well-defined kernel-mediated channels — signals for async notification, pipes for streaming data, shared memory for performance-critical paths. |
| `cgroups` and namespaces | These two Linux kernel features are the entire foundation of containers — cgroups limit resources, namespaces isolate visibility. |

---

## Layer 6 — Virtualisation & Containers

*How Docker and Kubernetes work, without the magic.*

### Topics

| Topic | Overview |
|---|---|
| Hardware virtualisation (hypervisors, VMs) | A hypervisor traps privileged instructions and emulates hardware so multiple OS kernels can run on one physical machine with near-native performance. |
| Linux namespaces (pid, net, mnt, uts, ipc, user) | Each namespace type hides a different slice of the system from a process — together they create the illusion of an isolated machine. |
| `cgroups` v2 for resource limits | cgroups enforce CPU, memory, and I/O budgets per process group — the mechanism behind `--memory` and `--cpus` in Docker. |
| OCI images and layers | A container image is a stack of read-only filesystem layers expressed as tarballs — the running container adds one writable layer on top. |
| Container runtime (runc, containerd) | containerd is the daemon; runc is the low-level binary that actually calls clone(2) and execve(2) to start a container process. |
| Virtual networking (veth pairs, bridges, overlay networks) | Each container gets a virtual Ethernet interface connected to a software bridge — packets cross namespace boundaries via veth pairs. |
| Kubernetes primitives (Pod, Service, Deployment) | A Pod is the scheduling unit (one or more containers sharing a network namespace), a Service is stable DNS/IP in front of dynamic Pods, a Deployment keeps a desired replica count. |

---

## Layer 7 — Network Protocols

*How data moves, in the order it actually moves.*

### Topics

| Topic | Overview |
|---|---|
| Ethernet & MAC addressing (L2) | Two machines on the same network segment exchange frames identified by 48-bit hardware addresses, resolved via ARP. |
| IP addressing & routing (L3) | IP provides best-effort packet delivery across networks using 32- or 128-bit addresses; routers forward packets hop-by-hop based on the destination prefix. |
| TCP: three-way handshake, flow control, congestion control | TCP converts an unreliable packet network into a reliable ordered byte stream by numbering every byte and requiring acknowledgement. |
| UDP: when and why | UDP sends datagrams with no delivery guarantee and no connection overhead — the right choice when latency matters more than reliability (DNS, video, games). |
| TLS / mutual TLS | TLS negotiates symmetric encryption on top of TCP using asymmetric crypto for key exchange and certificates for identity — the `s` in `https`. |
| DNS: resolution chain | A hostname is resolved by walking a hierarchy of authoritative servers, with aggressive caching at every step. |
| HTTP/1.1 → HTTP/2 → HTTP/3 (QUIC) | HTTP/1.1 serialises requests; HTTP/2 multiplexes streams over one TCP connection; HTTP/3 replaces TCP with QUIC to eliminate head-of-line blocking at the transport layer. |
| WebSockets & Server-Sent Events | WebSockets upgrade an HTTP connection to a full-duplex channel; SSE is a simpler one-way push over plain HTTP. |
| Load balancing (L4 vs L7) | L4 load balancers forward TCP connections; L7 load balancers inspect HTTP and can route, rewrite, and terminate TLS. |

---

## Bonus Layer — Security & Observability (2026 Essentials)

*Two topics the classic pyramid omits that are now table stakes.*

### Security

| Topic | Overview |
|---|---|
| Public-key cryptography & PKI | RSA and elliptic-curve crypto let two parties establish a shared secret without ever meeting — certificates bind a public key to an identity. |
| Common vulnerability classes (buffer overflow, injection, IDOR, SSRF) | Most real-world vulnerabilities are variations of three root causes: trusting input, confusing data and code, and broken access control. |
| Supply chain trust (SBOMs, signing, reproducible builds) | Modern attacks target your dependencies, not your code — knowing what you ship and verifying its provenance is now a baseline expectation. |
| OAuth 2 / OIDC | The standard protocol for delegated authorisation and federated identity — understanding the token flows prevents the most common auth bugs. |

### Observability

| Topic | Overview |
|---|---|
| Structured logging | Machine-readable JSON logs with consistent field names are the difference between `grep` and a query — crucial when you have more than one server. |
| Distributed tracing (OpenTelemetry) | A trace ID propagated through every service call lets you reconstruct the full path of a request across dozens of microservices. |
| Metrics & the RED method (Rate, Errors, Duration) | Three numbers per service tell you whether it is healthy — everything else is context you look at after these tell you something is wrong. |
| Alerting vs dashboards | Alerts should fire on symptoms (user-visible SLO breaches), not causes — dashboards are for diagnosis after an alert, not primary monitoring. |

---

## What this pyramid gives you

Once you've internalised these layers, the following things stop being black boxes:

- **Docker / Kubernetes** — namespaces + cgroups + overlay networking + the OCI image spec.
- **"Why is this slow?"** — cache misses, syscall overhead, GC pauses, head-of-line blocking.
- **Security incidents** — you know which layer was violated and roughly how.
- **Vendor claims** — you can sanity-check whether what is being promised is physically possible.
- **Any new tool** — "which layer does this live in?" is almost always enough to orient you.

> The goal is not encyclopaedic knowledge. It is the ability to form a hypothesis about where a problem lives, then go read the right documentation to confirm it.

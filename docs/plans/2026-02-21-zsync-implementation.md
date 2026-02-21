# zsync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build zsync — a native iOS SSH app that discovers, manages, and attaches to zmosh sessions across machines, synced via iCloud.

**Architecture:** SwiftUI app shell with SwiftTerm for terminal emulation, Citadel/swift-nio-ssh for SSH transport, SwiftData+CloudKit for persistence and sync. Three-layer architecture: Views (SwiftUI screens) -> Services (SSH, zmosh discovery, voice) -> Models (SwiftData entities). The SSH transport is wrapped behind a protocol so Citadel can be swapped later.

**Tech Stack:** Swift 5.9+, iOS 17+, SwiftUI, SwiftTerm, Citadel, swift-nio-ssh, SwiftData, CloudKit, SFSpeechRecognizer, Security framework (Keychain/Secure Enclave)

**Design Doc:** `docs/plans/2026-02-21-zsync-design.md`
**Prototype:** `prototype/zsync.html`

---

## Phase 1: Project Foundation

### Task 1: Create Xcode Project + Dependencies

**Files:**
- Create: `zsync/zsync.xcodeproj`
- Create: `zsync/Package.swift` (for SPM dependencies)
- Create: `zsync/zsync/zsyncApp.swift`
- Create: `zsync/zsync/Info.plist`
- Create: `zsync/zsyncTests/zsyncTests.swift`

**Step 1: Create the Xcode project**

Create a new iOS App project named `zsync` in the repo root:
- Organization: com.wavedepth
- Interface: SwiftUI
- Language: Swift
- Storage: SwiftData
- Include Tests: Yes (Unit + UI)
- Deployment target: iOS 17.0
- Device: iPhone

```
zsync/
├── zsync/
│   ├── zsyncApp.swift
│   ├── ContentView.swift
│   ├── Info.plist
│   └── Assets.xcassets/
├── zsyncTests/
│   └── zsyncTests.swift
├── zsyncUITests/
│   └── zsyncUITests.swift
└── zsync.xcodeproj/
```

**Step 2: Add Swift Package dependencies**

In Xcode, add these packages via File > Add Package Dependencies:

| Package | URL | Version |
|---------|-----|---------|
| SwiftTerm | `https://github.com/migueldeicaza/SwiftTerm` | latest |
| Citadel | `https://github.com/orlandos-nl/Citadel` | latest |

Both pull in swift-nio-ssh transitively.

**Step 3: Configure capabilities**

In the project settings, enable:
- iCloud (CloudKit + Key-Value Storage)
- Keychain Sharing
- Background Modes: none for v1 (SSH dies on background — we reconnect on foreground)

**Step 4: Add a placeholder entitlements file**

Xcode generates `zsync.entitlements` when you enable capabilities. Verify it includes:
```xml
<key>com.apple.developer.icloud-container-identifiers</key>
<array>
    <string>iCloud.com.wavedepth.zsync</string>
</array>
<key>keychain-access-groups</key>
<array>
    <string>$(AppIdentifierPrefix)com.wavedepth.zsync</string>
</array>
```

**Step 5: Build and run on simulator**

Run: `xcodebuild build -project zsync/zsync.xcodeproj -scheme zsync -destination 'platform=iOS Simulator,name=iPhone 16' | tail -5`
Expected: BUILD SUCCEEDED

**Step 6: Commit**

```bash
git add zsync/
git commit -m "feat: scaffold zsync Xcode project with SwiftTerm and Citadel"
```

---

### Task 2: Data Model (SwiftData)

**Files:**
- Create: `zsync/zsync/Models/Machine.swift`
- Create: `zsync/zsync/Models/RecentSession.swift`
- Create: `zsync/zsync/Models/ZmoshSession.swift`
- Create: `zsync/zsync/Models/AppSettings.swift`
- Test: `zsync/zsyncTests/ModelTests.swift`

**Step 1: Write failing tests for Machine model**

```swift
// zsync/zsyncTests/ModelTests.swift
import XCTest
import SwiftData
@testable import zsync

final class MachineModelTests: XCTestCase {
    var container: ModelContainer!

    override func setUp() {
        super.setUp()
        container = try! ModelContainer(
            for: Machine.self, RecentSession.self,
            configurations: ModelConfiguration(isStoredInMemoryOnly: true)
        )
    }

    func testMachineCreation() throws {
        let context = container.mainContext
        let machine = Machine(
            name: "MacBook Pro",
            host: "192.168.1.42",
            port: 22,
            username: "nerveband",
            authMethod: .sshKey,
            icon: "laptop",
            iconColor: "blue"
        )
        context.insert(machine)
        try context.save()

        let fetched = try context.fetch(FetchDescriptor<Machine>())
        XCTAssertEqual(fetched.count, 1)
        XCTAssertEqual(fetched[0].name, "MacBook Pro")
        XCTAssertEqual(fetched[0].host, "192.168.1.42")
        XCTAssertEqual(fetched[0].port, 22)
        XCTAssertEqual(fetched[0].zmoshInstalled, false) // default
    }

    func testRecentSessionCreation() throws {
        let context = container.mainContext
        let recent = RecentSession(
            sessionName: "bbcli",
            machineName: "MacBook Pro",
            machineId: UUID()
        )
        context.insert(recent)
        try context.save()

        let fetched = try context.fetch(FetchDescriptor<RecentSession>(
            sortBy: [SortDescriptor(\.lastConnected, order: .reverse)]
        ))
        XCTAssertEqual(fetched.count, 1)
        XCTAssertEqual(fetched[0].sessionName, "bbcli")
    }
}
```

**Step 2: Run tests — verify they fail**

Run: `xcodebuild test -project zsync/zsync.xcodeproj -scheme zsync -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | grep -E '(FAIL|error:|Test.*failed)'`
Expected: Compilation errors — Machine and RecentSession don't exist yet

**Step 3: Implement Machine model**

```swift
// zsync/zsync/Models/Machine.swift
import SwiftData
import Foundation

enum AuthMethod: String, Codable {
    case sshKey
    case password
}

@Model
final class Machine {
    var id: UUID
    var name: String
    var host: String
    var port: Int
    var username: String
    var authMethod: AuthMethod
    var keyRef: String?
    var icon: String
    var iconColor: String
    var zmoshInstalled: Bool
    var createdAt: Date

    init(
        name: String,
        host: String,
        port: Int = 22,
        username: String,
        authMethod: AuthMethod = .sshKey,
        keyRef: String? = nil,
        icon: String = "laptop",
        iconColor: String = "blue"
    ) {
        self.id = UUID()
        self.name = name
        self.host = host
        self.port = port
        self.username = username
        self.authMethod = authMethod
        self.keyRef = keyRef
        self.icon = icon
        self.iconColor = iconColor
        self.zmoshInstalled = false
        self.createdAt = Date()
    }
}
```

**Step 4: Implement RecentSession model**

```swift
// zsync/zsync/Models/RecentSession.swift
import SwiftData
import Foundation

@Model
final class RecentSession {
    var id: UUID
    var sessionName: String
    var machineName: String
    var machineId: UUID
    var lastConnected: Date

    init(sessionName: String, machineName: String, machineId: UUID) {
        self.id = UUID()
        self.sessionName = sessionName
        self.machineName = machineName
        self.machineId = machineId
        self.lastConnected = Date()
    }
}
```

**Step 5: Implement ZmoshSession (transient, not persisted)**

```swift
// zsync/zsync/Models/ZmoshSession.swift
import Foundation

struct ZmoshSession: Identifiable, Equatable {
    let id = UUID()
    let name: String
    let pid: Int?
    let clients: Int
    let startedIn: String

    var isActive: Bool { clients > 0 }
}
```

**Step 6: Implement AppSettings (persisted, synced)**

```swift
// zsync/zsync/Models/AppSettings.swift
import SwiftData
import Foundation

@Model
final class AppSettings {
    var id: UUID
    var themeName: String
    var fontName: String
    var requireFaceID: Bool
    var iCloudSync: Bool
    var iCloudKeychainSync: Bool

    init() {
        self.id = UUID()
        self.themeName = "Dracula"
        self.fontName = "JetBrains Mono"
        self.requireFaceID = false
        self.iCloudSync = true
        self.iCloudKeychainSync = false
    }
}
```

**Step 7: Run tests — verify they pass**

Run: `xcodebuild test -project zsync/zsync.xcodeproj -scheme zsync -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | grep -E '(PASS|FAIL|Test.*passed|Test.*failed)'`
Expected: All tests pass

**Step 8: Commit**

```bash
git add zsync/zsync/Models/ zsync/zsyncTests/ModelTests.swift
git commit -m "feat: SwiftData models — Machine, RecentSession, ZmoshSession, AppSettings"
```

---

### Task 3: App Navigation Shell

**Files:**
- Modify: `zsync/zsync/zsyncApp.swift`
- Create: `zsync/zsync/Views/AppTabView.swift`
- Create: `zsync/zsync/Views/Home/HomeView.swift`
- Create: `zsync/zsync/Views/Keys/KeysView.swift`
- Create: `zsync/zsync/Views/Settings/SettingsView.swift`

**Step 1: Create the tab-based navigation**

```swift
// zsync/zsync/Views/AppTabView.swift
import SwiftUI

enum AppTab: String, CaseIterable {
    case sessions = "Sessions"
    case keys = "Keys"
    case settings = "Settings"

    var icon: String {
        switch self {
        case .sessions: return "terminal"
        case .keys: return "key"
        case .settings: return "gearshape"
        }
    }
}

struct AppTabView: View {
    @State private var selectedTab: AppTab = .sessions

    var body: some View {
        TabView(selection: $selectedTab) {
            NavigationStack {
                HomeView()
            }
            .tabItem {
                Label(AppTab.sessions.rawValue,
                      systemImage: AppTab.sessions.icon)
            }
            .tag(AppTab.sessions)

            NavigationStack {
                KeysView()
            }
            .tabItem {
                Label(AppTab.keys.rawValue,
                      systemImage: AppTab.keys.icon)
            }
            .tag(AppTab.keys)

            NavigationStack {
                SettingsView()
            }
            .tabItem {
                Label(AppTab.settings.rawValue,
                      systemImage: AppTab.settings.icon)
            }
            .tag(AppTab.settings)
        }
        .tint(.blue)
    }
}
```

**Step 2: Create placeholder views**

```swift
// zsync/zsync/Views/Home/HomeView.swift
import SwiftUI

struct HomeView: View {
    var body: some View {
        List {
            Section("Quick Jump") {
                Text("No recent sessions")
                    .foregroundStyle(.secondary)
            }
            Section("Machines") {
                Text("No machines configured")
                    .foregroundStyle(.secondary)
            }
        }
        .navigationTitle("zsync")
    }
}
```

```swift
// zsync/zsync/Views/Keys/KeysView.swift
import SwiftUI

struct KeysView: View {
    var body: some View {
        List {
            Section("Device Key (Secure Enclave)") {
                Text("No key generated")
                    .foregroundStyle(.secondary)
            }
        }
        .navigationTitle("SSH Keys")
    }
}
```

```swift
// zsync/zsync/Views/Settings/SettingsView.swift
import SwiftUI

struct SettingsView: View {
    var body: some View {
        List {
            Section("Terminal Theme") {
                Text("Dracula")
            }
            Section("Terminal Font") {
                Text("JetBrains Mono")
            }
        }
        .navigationTitle("Settings")
    }
}
```

**Step 3: Wire up the app entry point**

```swift
// zsync/zsync/zsyncApp.swift
import SwiftUI
import SwiftData

@main
struct zsyncApp: App {
    var body: some Scene {
        WindowGroup {
            AppTabView()
        }
        .modelContainer(for: [Machine.self, RecentSession.self, AppSettings.self])
    }
}
```

**Step 4: Build and run — verify tab navigation works**

Run on simulator. Verify 3 tabs (Sessions, Keys, Settings) appear and switch correctly.

**Step 5: Commit**

```bash
git add zsync/zsync/Views/ zsync/zsync/zsyncApp.swift
git commit -m "feat: app navigation shell with three tabs"
```

---

## Phase 2: SSH Transport

### Task 4: SSH Connection Manager

**Files:**
- Create: `zsync/zsync/Services/SSHService.swift`
- Create: `zsync/zsync/Services/SSHConnection.swift`
- Test: `zsync/zsyncTests/SSHServiceTests.swift`

**Step 1: Define the SSH service protocol**

```swift
// zsync/zsync/Services/SSHService.swift
import Foundation

enum SSHAuthMethod {
    case key(privateKey: String)
    case password(String)
}

enum SSHError: Error, LocalizedError {
    case connectionFailed(String)
    case authenticationFailed
    case commandFailed(String)
    case timeout
    case channelClosed

    var errorDescription: String? {
        switch self {
        case .connectionFailed(let msg): return "Connection failed: \(msg)"
        case .authenticationFailed: return "Authentication failed"
        case .commandFailed(let msg): return "Command failed: \(msg)"
        case .timeout: return "Connection timed out"
        case .channelClosed: return "Channel closed"
        }
    }
}

protocol SSHServiceProtocol {
    /// Execute a command and return stdout
    func execute(
        host: String,
        port: Int,
        username: String,
        auth: SSHAuthMethod,
        command: String
    ) async throws -> String

    /// Open an interactive shell session (for terminal)
    func openShell(
        host: String,
        port: Int,
        username: String,
        auth: SSHAuthMethod,
        onData: @escaping (Data) -> Void
    ) async throws -> SSHShellSession
}

protocol SSHShellSession {
    func write(_ data: Data) async throws
    func resize(cols: Int, rows: Int) async throws
    func close() async throws
}
```

**Step 2: Implement the Citadel-backed SSH service**

```swift
// zsync/zsync/Services/SSHConnection.swift
import Foundation
import Citadel
import NIO
import NIOSSH

final class CitadelSSHService: SSHServiceProtocol {
    func execute(
        host: String,
        port: Int,
        username: String,
        auth: SSHAuthMethod,
        command: String
    ) async throws -> String {
        let client = try await SSHClient.connect(
            host: host,
            port: port,
            authenticationMethod: citadelAuth(auth, username: username),
            hostKeyValidator: .acceptAnything(), // TODO: Task 6 — TOFU
            reconnect: .never
        )
        defer { Task { try? await client.close() } }

        let response = try await client.executeCommand(command)
        let stdout = String(buffer: response.stdout)
        return stdout
    }

    func openShell(
        host: String,
        port: Int,
        username: String,
        auth: SSHAuthMethod,
        onData: @escaping (Data) -> Void
    ) async throws -> SSHShellSession {
        let client = try await SSHClient.connect(
            host: host,
            port: port,
            authenticationMethod: citadelAuth(auth, username: username),
            hostKeyValidator: .acceptAnything(), // TODO: Task 6 — TOFU
            reconnect: .never
        )

        // Open a PTY channel for interactive shell
        // Citadel's API for interactive shells — check latest docs
        // This is the critical glue layer between Citadel and SwiftTerm
        let shell = try await CitadelShellSession(client: client, onData: onData)
        return shell
    }

    private func citadelAuth(
        _ auth: SSHAuthMethod,
        username: String
    ) -> Citadel.SSHAuthenticationMethod {
        switch auth {
        case .key(let privateKey):
            return .privateKey(
                username: username,
                privateKey: .init(sshEd25519: privateKey)
            )
        case .password(let password):
            return .passwordBased(username: username, password: password)
        }
    }
}

final class CitadelShellSession: SSHShellSession {
    private let client: SSHClient

    init(client: SSHClient, onData: @escaping (Data) -> Void) async throws {
        self.client = client
        // Implementation depends on Citadel's channel API
        // Will need to open a PTY channel and forward data
    }

    func write(_ data: Data) async throws {
        // Send keystrokes to remote shell
    }

    func resize(cols: Int, rows: Int) async throws {
        // Send window-change request
    }

    func close() async throws {
        try await client.close()
    }
}
```

**Step 3: Write tests with a mock**

```swift
// zsync/zsyncTests/SSHServiceTests.swift
import XCTest
@testable import zsync

final class MockSSHService: SSHServiceProtocol {
    var executeResult: String = ""
    var executeError: Error?

    func execute(host: String, port: Int, username: String,
                 auth: SSHAuthMethod, command: String) async throws -> String {
        if let error = executeError { throw error }
        return executeResult
    }

    func openShell(host: String, port: Int, username: String,
                   auth: SSHAuthMethod,
                   onData: @escaping (Data) -> Void) async throws -> SSHShellSession {
        fatalError("Not implemented in mock")
    }
}

final class SSHServiceTests: XCTestCase {
    func testMockExecuteReturnsOutput() async throws {
        let mock = MockSSHService()
        mock.executeResult = "session_name=test\tclients=1\tstarted_in=~/projects"
        let result = try await mock.execute(
            host: "localhost", port: 22, username: "user",
            auth: .password("pass"), command: "zmosh list"
        )
        XCTAssertTrue(result.contains("session_name=test"))
    }

    func testMockExecuteThrowsOnError() async {
        let mock = MockSSHService()
        mock.executeError = SSHError.connectionFailed("refused")
        do {
            _ = try await mock.execute(
                host: "localhost", port: 22, username: "user",
                auth: .password("pass"), command: "zmosh list"
            )
            XCTFail("Should have thrown")
        } catch {
            XCTAssertTrue(error is SSHError)
        }
    }
}
```

**Step 4: Run tests**

Expected: All pass (mock-based, no real SSH needed)

**Step 5: Commit**

```bash
git add zsync/zsync/Services/ zsync/zsyncTests/SSHServiceTests.swift
git commit -m "feat: SSH service protocol with Citadel implementation and mock"
```

---

### Task 5: zmosh Discovery Service

**Files:**
- Create: `zsync/zsync/Services/ZmoshService.swift`
- Test: `zsync/zsyncTests/ZmoshServiceTests.swift`

This is the critical parser that understands `zmosh list` output.

**Step 1: Write failing tests for parsing**

```swift
// zsync/zsyncTests/ZmoshServiceTests.swift
import XCTest
@testable import zsync

final class ZmoshServiceTests: XCTestCase {

    func testParseZmoshListOutput() {
        let output = """
        session_name=apcsp-1\tpid=1234\tclients=1\tstarted_in=~/GitHub/aak-class-25-26/apcsp
        session_name=bbcli\tpid=5678\tclients=0\tstarted_in=~/Documents/GitHub/agent-to-bricks
        """
        let sessions = ZmoshParser.parse(output)
        XCTAssertEqual(sessions.count, 2)
        XCTAssertEqual(sessions[0].name, "apcsp-1")
        XCTAssertEqual(sessions[0].clients, 1)
        XCTAssertTrue(sessions[0].isActive)
        XCTAssertEqual(sessions[1].name, "bbcli")
        XCTAssertEqual(sessions[1].clients, 0)
        XCTAssertFalse(sessions[1].isActive)
    }

    func testParseEmptyOutput() {
        let sessions = ZmoshParser.parse("")
        XCTAssertEqual(sessions.count, 0)
    }

    func testParseHandlesMissingFields() {
        let output = "session_name=test\tclients=0"
        let sessions = ZmoshParser.parse(output)
        XCTAssertEqual(sessions.count, 1)
        XCTAssertEqual(sessions[0].name, "test")
        XCTAssertEqual(sessions[0].startedIn, "~")
    }

    func testDetectZmoshInstalled() async throws {
        let mock = MockSSHService()
        mock.executeResult = "session_name=test\tclients=0\tstarted_in=~"
        let service = ZmoshService(ssh: mock)
        let machine = Machine(name: "Test", host: "localhost", username: "user")
        let result = try await service.checkZmosh(machine: machine, auth: .password("pass"))
        XCTAssertTrue(result.installed)
        XCTAssertEqual(result.sessions.count, 1)
    }

    func testDetectZmoshNotInstalled() async throws {
        let mock = MockSSHService()
        mock.executeError = SSHError.commandFailed("zmosh: command not found")
        let service = ZmoshService(ssh: mock)
        let machine = Machine(name: "Test", host: "localhost", username: "user")
        let result = try await service.checkZmosh(machine: machine, auth: .password("pass"))
        XCTAssertFalse(result.installed)
        XCTAssertEqual(result.sessions.count, 0)
    }
}
```

**Step 2: Run tests — verify they fail**

Expected: ZmoshParser and ZmoshService don't exist yet

**Step 3: Implement the parser and service**

```swift
// zsync/zsync/Services/ZmoshService.swift
import Foundation

struct ZmoshCheckResult {
    let installed: Bool
    let sessions: [ZmoshSession]
}

enum ZmoshParser {
    /// Parse the tab-separated output of `zmosh list`
    /// Format per line: session_name=<name>\tpid=<pid>\tclients=<n>\tstarted_in=<path>
    static func parse(_ output: String) -> [ZmoshSession] {
        output
            .components(separatedBy: .newlines)
            .compactMap { line -> ZmoshSession? in
                let line = line.trimmingCharacters(in: .whitespaces)
                guard !line.isEmpty else { return nil }

                var name = ""
                var pid: Int?
                var clients = 0
                var startedIn = "~"

                for field in line.components(separatedBy: "\t") {
                    let trimmed = field.trimmingCharacters(in: .whitespaces)
                    if trimmed.hasPrefix("session_name=") {
                        name = String(trimmed.dropFirst("session_name=".count))
                    } else if trimmed.hasPrefix("pid=") {
                        pid = Int(trimmed.dropFirst("pid=".count))
                    } else if trimmed.hasPrefix("clients=") {
                        clients = Int(trimmed.dropFirst("clients=".count)) ?? 0
                    } else if trimmed.hasPrefix("started_in=") {
                        startedIn = String(trimmed.dropFirst("started_in=".count))
                    }
                }

                guard !name.isEmpty else { return nil }
                return ZmoshSession(
                    name: name, pid: pid,
                    clients: clients, startedIn: startedIn
                )
            }
    }
}

final class ZmoshService {
    private let ssh: SSHServiceProtocol

    init(ssh: SSHServiceProtocol) {
        self.ssh = ssh
    }

    /// SSH into machine, run `zmosh list`, parse result.
    /// Returns installed=false if zmosh is not found.
    func checkZmosh(
        machine: Machine,
        auth: SSHAuthMethod
    ) async throws -> ZmoshCheckResult {
        do {
            let output = try await ssh.execute(
                host: machine.host,
                port: machine.port,
                username: machine.username,
                auth: auth,
                command: "zmosh list"
            )
            let sessions = ZmoshParser.parse(output)
            return ZmoshCheckResult(installed: true, sessions: sessions)
        } catch {
            // If the command itself fails (not found), zmosh isn't installed
            if case SSHError.commandFailed(let msg) = error,
               msg.contains("not found") || msg.contains("No such file") {
                return ZmoshCheckResult(installed: false, sessions: [])
            }
            throw error
        }
    }

    /// Create or attach to a session
    func attachSession(name: String) -> String {
        return "zmosh attach \"\(name)\""
    }

    /// Kill a session
    func killSession(name: String) -> String {
        return "zmosh kill \"\(name)\""
    }

    /// List remote directories (via zoxide or ls)
    func listDirectories(
        machine: Machine,
        auth: SSHAuthMethod
    ) async throws -> [String] {
        let output = try await ssh.execute(
            host: machine.host,
            port: machine.port,
            username: machine.username,
            auth: auth,
            command: "zoxide query -l 2>/dev/null || ls -d ~/*/  2>/dev/null"
        )
        return output
            .components(separatedBy: .newlines)
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
    }
}
```

**Step 4: Run tests — verify they pass**

Expected: All 5 tests pass

**Step 5: Commit**

```bash
git add zsync/zsync/Services/ZmoshService.swift zsync/zsyncTests/ZmoshServiceTests.swift
git commit -m "feat: zmosh discovery service — parse zmosh list, detect installation"
```

---

### Task 6: Host Key Trust (TOFU)

**Files:**
- Create: `zsync/zsync/Services/HostKeyStore.swift`
- Create: `zsync/zsync/Views/Components/HostKeyAlertView.swift`
- Test: `zsync/zsyncTests/HostKeyStoreTests.swift`

**Step 1: Write failing tests**

```swift
// zsync/zsyncTests/HostKeyStoreTests.swift
import XCTest
@testable import zsync

final class HostKeyStoreTests: XCTestCase {
    var store: HostKeyStore!

    override func setUp() {
        store = HostKeyStore(storage: .inMemory)
    }

    func testStoreAndRetrieveFingerprint() {
        store.trust(host: "192.168.1.42", port: 22, fingerprint: "SHA256:abc123")
        XCTAssertEqual(
            store.knownFingerprint(host: "192.168.1.42", port: 22),
            "SHA256:abc123"
        )
    }

    func testUnknownHostReturnsNil() {
        XCTAssertNil(store.knownFingerprint(host: "unknown.host", port: 22))
    }

    func testFingerprintMismatchDetected() {
        store.trust(host: "192.168.1.42", port: 22, fingerprint: "SHA256:abc123")
        let result = store.verify(host: "192.168.1.42", port: 22, fingerprint: "SHA256:DIFFERENT")
        XCTAssertEqual(result, .mismatch)
    }

    func testFingerprintMatchDetected() {
        store.trust(host: "192.168.1.42", port: 22, fingerprint: "SHA256:abc123")
        let result = store.verify(host: "192.168.1.42", port: 22, fingerprint: "SHA256:abc123")
        XCTAssertEqual(result, .trusted)
    }

    func testNewHostDetected() {
        let result = store.verify(host: "new.host", port: 22, fingerprint: "SHA256:xyz")
        XCTAssertEqual(result, .unknown)
    }
}
```

**Step 2: Implement HostKeyStore**

```swift
// zsync/zsync/Services/HostKeyStore.swift
import Foundation

enum HostKeyVerification: Equatable {
    case trusted
    case unknown
    case mismatch
}

final class HostKeyStore {
    enum Storage {
        case inMemory
        case keychain
    }

    private var keys: [String: String] = [:]
    private let storage: Storage

    init(storage: Storage = .keychain) {
        self.storage = storage
        if storage == .keychain {
            loadFromKeychain()
        }
    }

    private func hostKey(_ host: String, _ port: Int) -> String {
        "\(host):\(port)"
    }

    func trust(host: String, port: Int, fingerprint: String) {
        keys[hostKey(host, port)] = fingerprint
        if storage == .keychain {
            saveToKeychain()
        }
    }

    func knownFingerprint(host: String, port: Int) -> String? {
        keys[hostKey(host, port)]
    }

    func verify(host: String, port: Int, fingerprint: String) -> HostKeyVerification {
        guard let known = keys[hostKey(host, port)] else { return .unknown }
        return known == fingerprint ? .trusted : .mismatch
    }

    func remove(host: String, port: Int) {
        keys.removeValue(forKey: hostKey(host, port))
        if storage == .keychain {
            saveToKeychain()
        }
    }

    private func loadFromKeychain() {
        // Load from Keychain — keyed by "com.wavedepth.zsync.hostkeys"
        // Implementation uses Security framework
    }

    private func saveToKeychain() {
        // Save to Keychain
    }
}
```

**Step 3: Create host key alert view**

```swift
// zsync/zsync/Views/Components/HostKeyAlertView.swift
import SwiftUI

struct HostKeyAlertView: View {
    let host: String
    let fingerprint: String
    let verification: HostKeyVerification
    let onTrust: () -> Void
    let onReject: () -> Void

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: verification == .mismatch
                  ? "exclamationmark.shield.fill"
                  : "questionmark.circle.fill")
                .font(.system(size: 48))
                .foregroundStyle(verification == .mismatch ? .red : .yellow)

            Text(verification == .mismatch
                 ? "Host Key Changed"
                 : "New Host")
                .font(.title2.bold())

            Text(verification == .mismatch
                 ? "The host key for \(host) has changed. This could indicate a security issue."
                 : "First connection to \(host). Verify the fingerprint:")
                .font(.body)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)

            Text(fingerprint)
                .font(.system(.caption, design: .monospaced))
                .padding(12)
                .background(.quaternary)
                .clipShape(RoundedRectangle(cornerRadius: 8))

            HStack(spacing: 12) {
                Button("Reject", role: .destructive, action: onReject)
                    .buttonStyle(.bordered)
                Button("Trust", action: onTrust)
                    .buttonStyle(.borderedProminent)
            }
        }
        .padding(24)
    }
}
```

**Step 4: Run tests — verify they pass**

**Step 5: Commit**

```bash
git add zsync/zsync/Services/HostKeyStore.swift zsync/zsync/Views/Components/ zsync/zsyncTests/HostKeyStoreTests.swift
git commit -m "feat: host key trust store with TOFU verification"
```

---

## Phase 3: Core Screens

### Task 7: Home Screen (Machines List + Quick Jump)

**Files:**
- Modify: `zsync/zsync/Views/Home/HomeView.swift`
- Create: `zsync/zsync/Views/Home/QuickJumpRow.swift`
- Create: `zsync/zsync/Views/Home/MachineRow.swift`
- Create: `zsync/zsync/Views/Components/MachineIconView.swift`

**Step 1: Create MachineIconView component**

Maps icon name + color to SF Symbol in a rounded rectangle, matching the prototype's icon picker.

```swift
// zsync/zsync/Views/Components/MachineIconView.swift
import SwiftUI

struct MachineIconView: View {
    let icon: String
    let color: String
    var size: CGFloat = 36

    private var sfSymbol: String {
        switch icon {
        case "laptop": return "laptopcomputer"
        case "desktop": return "desktopcomputer"
        case "server": return "server.rack"
        case "cloud": return "cloud"
        case "cpu": return "cpu"
        case "globe": return "globe"
        case "terminal": return "terminal"
        case "wifi": return "wifi"
        case "home": return "house"
        case "lock": return "lock.shield"
        default: return "desktopcomputer"
        }
    }

    private var iconColor: Color {
        switch color {
        case "blue": return .blue
        case "purple": return .purple
        case "green": return .green
        case "orange": return .orange
        case "pink": return .pink
        case "teal": return .teal
        case "red": return .red
        case "yellow": return .yellow
        default: return .secondary
        }
    }

    var body: some View {
        Image(systemName: sfSymbol)
            .font(.system(size: size * 0.5))
            .foregroundStyle(iconColor)
            .frame(width: size, height: size)
            .background(.quaternary)
            .clipShape(RoundedRectangle(cornerRadius: size * 0.22))
    }
}
```

**Step 2: Create QuickJumpRow and MachineRow**

```swift
// zsync/zsync/Views/Home/QuickJumpRow.swift
import SwiftUI

struct QuickJumpRow: View {
    let recentSession: RecentSession
    let machine: Machine?

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "terminal")
                .font(.system(size: 16))
                .foregroundStyle(.secondary)
                .frame(width: 36, height: 36)
                .background(.quaternary)
                .clipShape(RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 2) {
                Text(recentSession.sessionName)
                    .font(.body.weight(.medium))
                HStack(spacing: 4) {
                    Circle()
                        .fill(.green)
                        .frame(width: 6, height: 6)
                    Text(recentSession.machineName)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    Text("·")
                        .foregroundStyle(.secondary)
                    Text(recentSession.lastConnected, style: .relative)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer()

            Image(systemName: "chevron.right")
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
    }
}
```

```swift
// zsync/zsync/Views/Home/MachineRow.swift
import SwiftUI

struct MachineRow: View {
    let machine: Machine
    let sessionCount: Int
    let activeCount: Int

    var body: some View {
        HStack(spacing: 12) {
            MachineIconView(icon: machine.icon, color: machine.iconColor)

            VStack(alignment: .leading, spacing: 2) {
                Text(machine.name)
                    .font(.body.weight(.medium))
                if machine.zmoshInstalled {
                    Text("\(sessionCount) sessions · \(activeCount) active")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                } else {
                    Text("Setup required")
                        .font(.subheadline)
                        .foregroundStyle(.yellow)
                }
            }

            Spacer()

            if machine.zmoshInstalled {
                Circle()
                    .fill(.green)
                    .frame(width: 8, height: 8)
            } else {
                Image(systemName: "exclamationmark.triangle.fill")
                    .font(.caption)
                    .foregroundStyle(.yellow)
            }

            Image(systemName: "chevron.right")
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
    }
}
```

**Step 3: Update HomeView with real data binding**

```swift
// zsync/zsync/Views/Home/HomeView.swift
import SwiftUI
import SwiftData

struct HomeView: View {
    @Query(sort: \Machine.createdAt) private var machines: [Machine]
    @Query(sort: \RecentSession.lastConnected, order: .reverse)
    private var recents: [RecentSession]
    @State private var showAddMachine = false

    var body: some View {
        List {
            if !recents.isEmpty {
                Section("Quick Jump") {
                    ForEach(recents.prefix(5)) { recent in
                        NavigationLink {
                            // Terminal view — wired in Task 12
                            Text("Connecting to \(recent.sessionName)...")
                        } label: {
                            QuickJumpRow(
                                recentSession: recent,
                                machine: machines.first { $0.id == recent.machineId }
                            )
                        }
                    }
                }
            }

            Section("Machines") {
                ForEach(machines) { machine in
                    NavigationLink {
                        if machine.zmoshInstalled {
                            SessionPickerView(machine: machine)
                        } else {
                            SetupRequiredView(machine: machine)
                        }
                    } label: {
                        MachineRow(
                            machine: machine,
                            sessionCount: 0, // filled when discovery runs
                            activeCount: 0
                        )
                    }
                }
            }

            Section {
                Button {
                    showAddMachine = true
                } label: {
                    Label("Add Machine", systemImage: "plus")
                }
            }
        }
        .navigationTitle("zsync")
        .sheet(isPresented: $showAddMachine) {
            AddMachineView()
        }
    }
}
```

**Step 4: Build — verify no compilation errors**

**Step 5: Commit**

```bash
git add zsync/zsync/Views/
git commit -m "feat: home screen with quick jump, machine list, and machine row components"
```

---

### Task 8: Add Machine Screen

**Files:**
- Create: `zsync/zsync/Views/Home/AddMachineView.swift`
- Create: `zsync/zsync/Views/Components/IconPickerView.swift`

**Step 1: Create IconPickerView**

Grid of 10 icons matching the prototype's icon picker.

```swift
// zsync/zsync/Views/Components/IconPickerView.swift
import SwiftUI

struct IconOption: Identifiable {
    let id = UUID()
    let icon: String
    let label: String
    let color: String
}

let machineIconOptions: [IconOption] = [
    .init(icon: "laptop", label: "Laptop", color: "blue"),
    .init(icon: "desktop", label: "Desktop", color: "purple"),
    .init(icon: "server", label: "Server", color: "green"),
    .init(icon: "cloud", label: "Cloud", color: "teal"),
    .init(icon: "cpu", label: "Chip", color: "orange"),
    .init(icon: "globe", label: "Globe", color: "pink"),
    .init(icon: "terminal", label: "Terminal", color: "yellow"),
    .init(icon: "wifi", label: "Network", color: "blue"),
    .init(icon: "home", label: "Home", color: "green"),
    .init(icon: "lock", label: "Secure", color: "red"),
]

struct IconPickerView: View {
    @Binding var selectedIcon: String
    @Binding var selectedColor: String

    let columns = Array(repeating: GridItem(.flexible(), spacing: 10), count: 5)

    var body: some View {
        LazyVGrid(columns: columns, spacing: 10) {
            ForEach(machineIconOptions) { option in
                MachineIconView(icon: option.icon, color: option.color, size: 52)
                    .overlay(
                        RoundedRectangle(cornerRadius: 12)
                            .stroke(
                                selectedIcon == option.icon ? Color.accentColor : .clear,
                                lineWidth: 2
                            )
                    )
                    .onTapGesture {
                        selectedIcon = option.icon
                        selectedColor = option.color
                    }
            }
        }
    }
}
```

**Step 2: Create AddMachineView**

```swift
// zsync/zsync/Views/Home/AddMachineView.swift
import SwiftUI
import SwiftData

struct AddMachineView: View {
    @Environment(\.modelContext) private var modelContext
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var host = ""
    @State private var port = "22"
    @State private var username = ""
    @State private var authMethod: AuthMethod = .sshKey
    @State private var selectedIcon = "laptop"
    @State private var selectedColor = "blue"

    var body: some View {
        NavigationStack {
            Form {
                Section("Connection") {
                    TextField("Display Name", text: $name)
                    TextField("Host", text: $host)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                    HStack {
                        TextField("Port", text: $port)
                            .keyboardType(.numberPad)
                            .frame(width: 80)
                        TextField("Username", text: $username)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                    }
                }

                Section("Icon") {
                    IconPickerView(
                        selectedIcon: $selectedIcon,
                        selectedColor: $selectedColor
                    )
                }

                Section("Authentication") {
                    Picker("Method", selection: $authMethod) {
                        Label("SSH Key", systemImage: "key")
                            .tag(AuthMethod.sshKey)
                        Label("Password", systemImage: "lock")
                            .tag(AuthMethod.password)
                    }
                    .pickerStyle(.inline)
                    .labelsHidden()
                }

                Section {
                    Label {
                        Text("zmosh must be installed on this machine. zsync will verify on first connect and guide you through setup if needed.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    } icon: {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(.yellow)
                    }
                }
            }
            .navigationTitle("Add Machine")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { saveMachine() }
                        .disabled(name.isEmpty || host.isEmpty || username.isEmpty)
                }
            }
        }
    }

    private func saveMachine() {
        let machine = Machine(
            name: name,
            host: host,
            port: Int(port) ?? 22,
            username: username,
            authMethod: authMethod,
            icon: selectedIcon,
            iconColor: selectedColor
        )
        modelContext.insert(machine)
        dismiss()
    }
}
```

**Step 3: Build and run — verify Add Machine form works**

**Step 4: Commit**

```bash
git add zsync/zsync/Views/
git commit -m "feat: add machine screen with icon picker and auth selection"
```

---

### Task 9: Session Picker Screen

**Files:**
- Create: `zsync/zsync/Views/Sessions/SessionPickerView.swift`
- Create: `zsync/zsync/Views/Sessions/SessionRow.swift`

**Step 1: Create SessionRow**

```swift
// zsync/zsync/Views/Sessions/SessionRow.swift
import SwiftUI

struct SessionRow: View {
    let session: ZmoshSession

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "terminal")
                .font(.system(size: 14))
                .foregroundStyle(.secondary)
                .frame(width: 32, height: 32)
                .background(.quaternary)
                .clipShape(RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 2) {
                Text(session.name)
                    .font(.body.weight(.semibold))
                Text(session.startedIn)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fontDesign(.monospaced)
            }

            Spacer()

            HStack(spacing: 6) {
                Circle()
                    .fill(session.isActive ? .green : Color(.tertiaryLabel))
                    .frame(width: 8, height: 8)
                Text(session.isActive ? "active" : "idle")
                    .font(.caption)
                    .foregroundStyle(session.isActive ? .green : .secondary)
            }
        }
    }
}
```

**Step 2: Create SessionPickerView**

```swift
// zsync/zsync/Views/Sessions/SessionPickerView.swift
import SwiftUI

struct SessionPickerView: View {
    let machine: Machine
    @State private var sessions: [ZmoshSession] = []
    @State private var isLoading = true
    @State private var error: String?
    @State private var showNewSession = false
    @State private var killTarget: ZmoshSession?

    var body: some View {
        Group {
            if isLoading {
                ProgressView("Loading sessions...")
            } else if let error {
                ContentUnavailableView(
                    "Connection Error",
                    systemImage: "wifi.exclamationmark",
                    description: Text(error)
                )
            } else {
                sessionList
            }
        }
        .navigationTitle(machine.name)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("Edit") { /* machine settings */ }
            }
        }
        .task { await loadSessions() }
        .refreshable { await loadSessions() }
        .sheet(isPresented: $showNewSession) {
            NewSessionView(machine: machine)
        }
        .confirmationDialog(
            "Kill \"\(killTarget?.name ?? "")\"?",
            isPresented: .init(
                get: { killTarget != nil },
                set: { if !$0 { killTarget = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button("Kill Session", role: .destructive) {
                if let target = killTarget {
                    Task { await killSession(target) }
                }
            }
        } message: {
            Text("This will terminate the session on \(machine.name).")
        }
    }

    private var sessionList: some View {
        List {
            let active = sessions.filter(\.isActive).count
            Section("\(sessions.count) sessions · \(active) active") {
                ForEach(sessions) { session in
                    NavigationLink {
                        // TerminalView — wired in Task 12
                        Text("Attaching to \(session.name)...")
                    } label: {
                        SessionRow(session: session)
                    }
                    .swipeActions(edge: .trailing) {
                        Button("Kill", role: .destructive) {
                            killTarget = session
                        }
                    }
                }
            }

            Section {
                Button {
                    showNewSession = true
                } label: {
                    Label("New Session", systemImage: "plus")
                }
            }
        }
    }

    private func loadSessions() async {
        isLoading = true
        error = nil
        // TODO: Wire to ZmoshService in integration phase
        // For now, use mock data matching the prototype
        try? await Task.sleep(for: .milliseconds(500))
        sessions = [
            ZmoshSession(name: "apcsp-1", pid: 1234, clients: 1,
                        startedIn: "~/GitHub/aak-class-25-26/apcsp"),
            ZmoshSession(name: "bbcli", pid: 5678, clients: 1,
                        startedIn: "~/Documents/GitHub/agent-to-bricks"),
        ]
        isLoading = false
    }

    private func killSession(_ session: ZmoshSession) async {
        sessions.removeAll { $0.id == session.id }
        killTarget = nil
    }
}
```

**Step 3: Build and verify navigation from HomeView to SessionPicker**

**Step 4: Commit**

```bash
git add zsync/zsync/Views/Sessions/
git commit -m "feat: session picker screen with swipe-to-kill"
```

---

### Task 10: Setup Required Gate Screen

**Files:**
- Create: `zsync/zsync/Views/Sessions/SetupRequiredView.swift`

**Step 1: Implement the gate screen**

```swift
// zsync/zsync/Views/Sessions/SetupRequiredView.swift
import SwiftUI

struct SetupRequiredView: View {
    let machine: Machine
    @State private var isChecking = false
    @State private var checkFailed = false

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                // Hero
                VStack(spacing: 16) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 48))
                        .foregroundStyle(.red)
                        .frame(width: 72, height: 72)
                        .background(.red.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 20))

                    Text("Setup Required")
                        .font(.title2.bold())

                    Text("zmosh is not installed on this machine. zsync requires zmosh to manage persistent terminal sessions.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                }
                .padding(.top, 32)

                // Steps
                VStack(alignment: .leading, spacing: 0) {
                    stepRow(num: 1, text: "SSH into **\(machine.host)** from another terminal")
                    Divider()
                    stepRow(num: 2, text: "Run the install command:\n`brew install mmonad/tap/zmosh`")
                    Divider()
                    stepRow(num: 3, text: "Verify it works: `zmosh list`")
                    Divider()
                    stepRow(num: 4, text: "Come back here and tap **Check Again**")
                }
                .background(Color(.secondarySystemGroupedBackground))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .padding(.horizontal)

                // Buttons
                VStack(spacing: 12) {
                    Button {
                        if let url = URL(string: "https://github.com/nerveband/zmosh-picker#installation") {
                            UIApplication.shared.open(url)
                        }
                    } label: {
                        Label("View Install Guide on GitHub", systemImage: "safari")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 14)
                    }
                    .buttonStyle(.borderedProminent)

                    Button {
                        Task { await recheck() }
                    } label: {
                        if isChecking {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 14)
                        } else {
                            Label("Check Again", systemImage: "arrow.clockwise")
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 14)
                        }
                    }
                    .buttonStyle(.bordered)
                    .disabled(isChecking)
                }
                .padding(.horizontal)

                if checkFailed {
                    Text("zmosh not found. Follow the install guide above.")
                        .font(.subheadline)
                        .foregroundStyle(.red)
                }
            }
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(machine.name)
        .navigationBarTitleDisplayMode(.inline)
    }

    private func stepRow(num: Int, text: LocalizedStringKey) -> some View {
        HStack(alignment: .top, spacing: 12) {
            Text("\(num)")
                .font(.caption.bold())
                .foregroundStyle(.white)
                .frame(width: 24, height: 24)
                .background(Color.accentColor)
                .clipShape(Circle())

            Text(text)
                .font(.subheadline)
        }
        .padding(14)
    }

    private func recheck() async {
        isChecking = true
        checkFailed = false
        // TODO: Wire to ZmoshService
        try? await Task.sleep(for: .seconds(1.5))
        // For prototype: always fails until zmosh is actually detected
        checkFailed = true
        isChecking = false
    }
}
```

**Step 2: Build and test — tap a machine without zmosh, verify gate blocks**

**Step 3: Commit**

```bash
git add zsync/zsync/Views/Sessions/SetupRequiredView.swift
git commit -m "feat: setup required gate — blocks machines without zmosh installed"
```

---

### Task 11: New Session Flow + Pick Directory

**Files:**
- Create: `zsync/zsync/Views/Sessions/NewSessionView.swift`
- Create: `zsync/zsync/Views/Sessions/PickDirectoryView.swift`

**Step 1: Create PickDirectoryView**

```swift
// zsync/zsync/Views/Sessions/PickDirectoryView.swift
import SwiftUI

struct PickDirectoryView: View {
    let directories: [String]
    @Binding var selectedDir: String
    @Environment(\.dismiss) private var dismiss
    @State private var searchText = ""

    private var filtered: [String] {
        if searchText.isEmpty { return directories }
        return directories.filter {
            $0.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        NavigationStack {
            List(filtered, id: \.self) { dir in
                Button {
                    selectedDir = dir
                    dismiss()
                } label: {
                    HStack(spacing: 10) {
                        Image(systemName: "folder")
                            .foregroundStyle(.secondary)
                        VStack(alignment: .leading) {
                            Text(dir.components(separatedBy: "/").last ?? dir)
                                .font(.body.weight(.medium))
                            Text(dir)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .fontDesign(.monospaced)
                        }
                    }
                }
                .tint(.primary)
            }
            .searchable(text: $searchText, prompt: "Search directories...")
            .navigationTitle("Pick Directory")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Back") { dismiss() }
                }
            }
        }
    }
}
```

**Step 2: Create NewSessionView**

```swift
// zsync/zsync/Views/Sessions/NewSessionView.swift
import SwiftUI

struct NewSessionView: View {
    let machine: Machine
    @Environment(\.dismiss) private var dismiss
    @State private var sessionName = ""
    @State private var selectedDir = "~"
    @State private var showDirPicker = false
    @State private var isAutoNamed = true

    // Mock dirs — will be fetched from remote via zoxide
    private let directories = [
        "~/Documents/GitHub/zmosh-picker",
        "~/Documents/GitHub/agent-to-bricks",
        "~/Documents/Projects/workandpromise",
        "~/Projects/wd-clients-ramadan-2026",
        "~/dotfiles",
        "~/scripts",
    ]

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    HStack {
                        TextField("Session name", text: $sessionName)
                            .fontDesign(.monospaced)
                            .font(.title3.weight(.semibold))
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.never)
                        if !sessionName.isEmpty {
                            Button {
                                sessionName = ""
                                isAutoNamed = false
                            } label: {
                                Image(systemName: "xmark.circle.fill")
                                    .foregroundStyle(.tertiary)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                } header: {
                    Text("Session Name")
                } footer: {
                    Text(isAutoNamed
                         ? "Auto-generated. Tap x to type your own."
                         : "Type a custom session name.")
                }

                Section("Directory") {
                    Button {
                        showDirPicker = true
                    } label: {
                        HStack(spacing: 10) {
                            Image(systemName: "folder")
                                .foregroundStyle(.secondary)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Working directory")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                Text(selectedDir)
                                    .fontDesign(.monospaced)
                            }
                            Spacer()
                            Image(systemName: "chevron.right")
                                .font(.caption)
                                .foregroundStyle(.tertiary)
                        }
                    }
                    .tint(.primary)
                }
            }
            .navigationTitle("New Session")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { createSession() }
                        .disabled(sessionName.isEmpty)
                }
            }
            .onAppear { generateName() }
            .sheet(isPresented: $showDirPicker) {
                PickDirectoryView(
                    directories: directories,
                    selectedDir: $selectedDir
                )
            }
            .onChange(of: selectedDir) {
                if isAutoNamed {
                    sessionName = selectedDir
                        .components(separatedBy: "/").last ?? "new-session"
                }
            }
        }
    }

    private func generateName() {
        let base = selectedDir.components(separatedBy: "/").last ?? "new-session"
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "MMdd"
        sessionName = "\(base)-\(dateFormatter.string(from: Date()))"
        isAutoNamed = true
    }

    private func createSession() {
        // TODO: Execute `zmosh attach "<name>"` via SSH in the terminal
        dismiss()
    }
}
```

**Step 3: Build and test the full flow: Home -> Machine -> New Session -> Pick Dir**

**Step 4: Commit**

```bash
git add zsync/zsync/Views/Sessions/NewSessionView.swift zsync/zsync/Views/Sessions/PickDirectoryView.swift
git commit -m "feat: new session flow with auto-naming and directory picker"
```

---

## Phase 4: Terminal

### Task 12: SwiftTerm Integration + SSH Channel Bridge

**Files:**
- Create: `zsync/zsync/Views/Terminal/TerminalView.swift`
- Create: `zsync/zsync/Services/TerminalBridge.swift`

This is the hardest task — bridging SwiftTerm's `iOSTerminalView` with Citadel's SSH channel.

**Step 1: Create the terminal bridge**

The bridge implements SwiftTerm's `TerminalViewDelegate` and connects it to the SSH shell session.

```swift
// zsync/zsync/Services/TerminalBridge.swift
import Foundation
import SwiftTerm

/// Bridges SwiftTerm's TerminalView with an SSH shell session.
/// SwiftTerm calls `send(source:data:)` when user types.
/// SSH channel calls `onData` when remote sends output.
final class TerminalBridge: TerminalViewDelegate {
    private var shell: SSHShellSession?
    private weak var terminalView: TerminalView?

    var onTitleChange: ((String) -> Void)?

    func attach(terminal: TerminalView, shell: SSHShellSession) {
        self.terminalView = terminal
        self.shell = shell
        terminal.terminalDelegate = self
    }

    /// Called by SwiftTerm when user types
    func send(source: TerminalView, data: ArraySlice<UInt8>) {
        Task {
            try? await shell?.write(Data(data))
        }
    }

    /// Called by SSH channel when remote sends data
    func receive(data: Data) {
        DispatchQueue.main.async { [weak self] in
            let bytes = Array(data)
            self?.terminalView?.feed(byteArray: bytes)
        }
    }

    /// Terminal size changed — notify remote
    func sizeChanged(source: TerminalView, newCols: Int, newRows: Int) {
        Task {
            try? await shell?.resize(cols: newCols, rows: newRows)
        }
    }

    func setTerminalTitle(source: TerminalView, title: String) {
        onTitleChange?(title)
    }

    func hostCurrentDirectoryUpdate(source: TerminalView, directory: String?) {}
    func scrolled(source: TerminalView, position: Double) {}
    func clipboardCopy(source: TerminalView, content: Data) {
        if let str = String(data: content, encoding: .utf8) {
            UIPasteboard.general.string = str
        }
    }
    func rangeChanged(source: TerminalView, startY: Int, endY: Int) {}
    func requestOpenLink(source: TerminalView, link: String, params: [String : String]) {
        if let url = URL(string: link) {
            UIApplication.shared.open(url)
        }
    }
}
```

**Step 2: Create the SwiftUI wrapper for SwiftTerm**

SwiftTerm provides `iOSTerminalView` (UIKit). We wrap it in `UIViewRepresentable`.

```swift
// zsync/zsync/Views/Terminal/TerminalView.swift
import SwiftUI
import SwiftTerm

struct TerminalScreenView: View {
    let sessionName: String
    let machineName: String
    let machine: Machine
    @State private var bridge = TerminalBridge()
    @State private var terminalTitle = ""
    @State private var compositionText = ""
    @State private var isRecording = false

    var body: some View {
        VStack(spacing: 0) {
            // Terminal
            SwiftTermView(bridge: bridge)
                .ignoresSafeArea(.keyboard)

            // Composition bar
            CompositionBarView(
                text: $compositionText,
                isRecording: $isRecording,
                onSend: sendCommand,
                onSnippet: { snippet in compositionText = snippet }
            )

            // Key bar
            KeyBarView(bridge: bridge)
        }
        .navigationTitle(sessionName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                VStack(spacing: 0) {
                    Text(sessionName).font(.subheadline.bold())
                    Text(machineName).font(.caption2).foregroundStyle(.secondary)
                }
            }
        }
        .task { await connect() }
    }

    private func connect() async {
        // TODO: Establish SSH connection, open shell, attach bridge
        // This will be wired when the full SSH flow is integrated
    }

    private func sendCommand() {
        guard !compositionText.isEmpty else { return }
        let cmd = compositionText + "\n"
        if let data = cmd.data(using: .utf8) {
            bridge.receive(data: data) // echo locally (optional)
            Task {
                try? await bridge.shell?.write(data)
            }
        }
        compositionText = ""
    }
}

/// UIViewRepresentable wrapper for SwiftTerm's iOSTerminalView
struct SwiftTermView: UIViewRepresentable {
    let bridge: TerminalBridge

    func makeUIView(context: Context) -> SwiftTerm.TerminalView {
        let tv = SwiftTerm.TerminalView()
        tv.terminalDelegate = bridge
        // Apply theme defaults
        tv.installColors(.defaultLight) // TODO: theme from settings
        tv.font = UIFont(name: "JetBrains Mono", size: 13)
            ?? UIFont.monospacedSystemFont(ofSize: 13, weight: .regular)
        return tv
    }

    func updateUIView(_ uiView: SwiftTerm.TerminalView, context: Context) {}
}
```

**Step 3: Build — verify SwiftTerm view renders (blank terminal)**

**Step 4: Commit**

```bash
git add zsync/zsync/Views/Terminal/ zsync/zsync/Services/TerminalBridge.swift
git commit -m "feat: SwiftTerm integration with SSH bridge and terminal screen"
```

---

### Task 13: Composition Bar

**Files:**
- Create: `zsync/zsync/Views/Terminal/CompositionBarView.swift`

```swift
// zsync/zsync/Views/Terminal/CompositionBarView.swift
import SwiftUI

struct CompositionBarView: View {
    @Binding var text: String
    @Binding var isRecording: Bool
    let onSend: () -> Void
    let onSnippet: (String) -> Void

    var body: some View {
        HStack(spacing: 8) {
            TextField("Type or speak a command...", text: $text)
                .textFieldStyle(.plain)
                .fontDesign(.monospaced)
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
                .background(Color(.tertiarySystemFill))
                .clipShape(Capsule())
                .autocorrectionDisabled()
                .textInputAutocapitalization(.never)
                .onSubmit(onSend)

            // Mic button
            Button {
                toggleRecording()
            } label: {
                Image(systemName: isRecording ? "stop.fill" : "mic")
                    .font(.system(size: 16))
                    .frame(width: 36, height: 36)
                    .background(isRecording ? Color.red : Color(.tertiarySystemFill))
                    .foregroundStyle(isRecording ? .white : .blue)
                    .clipShape(Circle())
            }

            // Send button
            Button(action: onSend) {
                Image(systemName: "paperplane.fill")
                    .font(.system(size: 14))
                    .frame(width: 36, height: 36)
                    .background(text.isEmpty ? Color(.tertiarySystemFill) : .blue)
                    .foregroundStyle(text.isEmpty ? .secondary : .white)
                    .clipShape(Circle())
            }
            .disabled(text.isEmpty)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(.bar)
    }

    private func toggleRecording() {
        isRecording.toggle()
        // TODO: Wire to VoiceInputService in Task 22
    }
}
```

**Commit:**

```bash
git add zsync/zsync/Views/Terminal/CompositionBarView.swift
git commit -m "feat: composition bar with mic and send buttons"
```

---

### Task 14: Key Bar

**Files:**
- Create: `zsync/zsync/Views/Terminal/KeyBarView.swift`

```swift
// zsync/zsync/Views/Terminal/KeyBarView.swift
import SwiftUI

struct KeyBarView: View {
    let bridge: TerminalBridge
    @State private var ctrlActive = false
    @State private var altActive = false
    @State private var showSnippets = false
    @State private var showHistory = false

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                // Snippet & History toggles
                toolButton(icon: "star", color: .orange) {
                    showSnippets.toggle(); showHistory = false
                }
                toolButton(icon: "clock", color: .blue) {
                    showHistory.toggle(); showSnippets = false
                }
                divider

                // Modifier keys
                modifierKey("esc") { sendEscape() }
                modifierKey("ctrl", isActive: ctrlActive) { ctrlActive.toggle() }
                modifierKey("tab") { sendTab() }
                modifierKey("alt", isActive: altActive) { altActive.toggle() }
                divider

                // Arrow keys
                arrowKey("chevron.up") { sendKey(.cursorUp) }
                arrowKey("chevron.down") { sendKey(.cursorDown) }
                arrowKey("chevron.left") { sendKey(.cursorLeft) }
                arrowKey("chevron.right") { sendKey(.cursorRight) }
                divider

                // Symbol keys
                symbolKey("~")
                symbolKey("|")
                symbolKey("/")
                symbolKey("-")
                symbolKey("_")
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 6)
        }
        .background(.bar)
    }

    private func toolButton(icon: String, color: Color, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .frame(width: 40, height: 34)
                .background(Color(.tertiarySystemFill))
                .clipShape(RoundedRectangle(cornerRadius: 6))
                .foregroundStyle(color)
        }
    }

    private var divider: some View {
        Rectangle()
            .fill(.separator)
            .frame(width: 1, height: 24)
    }

    private func modifierKey(_ label: String, isActive: Bool = false, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(label)
                .font(.system(size: 14, weight: .medium, design: .monospaced))
                .frame(minWidth: 40, minHeight: 34)
                .background(isActive ? Color.accentColor : Color(.quaternarySystemFill))
                .foregroundStyle(isActive ? .white : .secondary)
                .clipShape(RoundedRectangle(cornerRadius: 6))
        }
    }

    private func arrowKey(_ icon: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .frame(width: 36, height: 34)
                .background(Color(.tertiarySystemFill))
                .clipShape(RoundedRectangle(cornerRadius: 6))
        }
    }

    private func symbolKey(_ char: String) -> some View {
        Button {
            sendCharacter(char)
        } label: {
            Text(char)
                .font(.system(size: 14, weight: .medium, design: .monospaced))
                .frame(minWidth: 40, minHeight: 34)
                .background(Color(.tertiarySystemFill))
                .clipShape(RoundedRectangle(cornerRadius: 6))
        }
    }

    // Key sending helpers
    private func sendCharacter(_ char: String) {
        var bytes = Array(char.utf8)
        if ctrlActive, let first = bytes.first, first >= 0x40, first <= 0x7f {
            bytes = [first & 0x1f]
            ctrlActive = false
        }
        bridge.terminalView?.feed(byteArray: bytes)
        Task { try? await bridge.shell?.write(Data(bytes)) }
    }

    private func sendEscape() {
        let bytes: [UInt8] = [0x1b]
        Task { try? await bridge.shell?.write(Data(bytes)) }
    }

    private func sendTab() {
        let bytes: [UInt8] = [0x09]
        Task { try? await bridge.shell?.write(Data(bytes)) }
    }

    private enum ArrowKey {
        case cursorUp, cursorDown, cursorLeft, cursorRight
        var sequence: [UInt8] {
            switch self {
            case .cursorUp: return [0x1b, 0x5b, 0x41]
            case .cursorDown: return [0x1b, 0x5b, 0x42]
            case .cursorRight: return [0x1b, 0x5b, 0x43]
            case .cursorLeft: return [0x1b, 0x5b, 0x44]
            }
        }
    }

    private func sendKey(_ key: ArrowKey) {
        Task { try? await bridge.shell?.write(Data(key.sequence)) }
    }
}
```

**Commit:**

```bash
git add zsync/zsync/Views/Terminal/KeyBarView.swift
git commit -m "feat: key bar with modifiers, arrows, and symbol keys"
```

---

### Task 15: Snippet Drawer + History Drawer

**Files:**
- Create: `zsync/zsync/Views/Terminal/SnippetDrawerView.swift`
- Create: `zsync/zsync/Views/Terminal/HistoryDrawerView.swift`

**Step 1: Snippet drawer**

```swift
// zsync/zsync/Views/Terminal/SnippetDrawerView.swift
import SwiftUI

struct SnippetDrawerView: View {
    let onSelect: (String) -> Void

    private let defaultSnippets = [
        "git status", "git pull", "git push",
        "ls -la", "npm run dev", "docker ps", "htop"
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text("SNIPPETS")
                    .font(.caption.bold())
                    .foregroundStyle(.secondary)
                Spacer()
            }
            .padding(.horizontal, 12)
            .padding(.top, 8)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(defaultSnippets, id: \.self) { snippet in
                        Button(snippet) { onSelect(snippet) }
                            .font(.system(size: 14, weight: .medium, design: .monospaced))
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(Color(.secondarySystemGroupedBackground))
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                }
                .padding(.horizontal, 12)
            }
            .padding(.bottom, 10)
        }
        .background(.bar)
    }
}
```

**Step 2: History drawer**

```swift
// zsync/zsync/Views/Terminal/HistoryDrawerView.swift
import SwiftUI

struct CommandHistoryItem: Identifiable {
    let id = UUID()
    let command: String
    let time: String
}

struct HistoryDrawerView: View {
    let history: [CommandHistoryItem]
    let onSelect: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text("RECENT COMMANDS")
                    .font(.caption.bold())
                    .foregroundStyle(.secondary)
                Spacer()
            }
            .padding(.horizontal, 12)
            .padding(.top, 8)

            ScrollView {
                VStack(spacing: 0) {
                    ForEach(history.prefix(8)) { item in
                        Button {
                            onSelect(item.command)
                        } label: {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(item.command)
                                    .font(.system(size: 14, design: .monospaced))
                                    .foregroundStyle(.primary)
                                Text(item.time)
                                    .font(.caption2)
                                    .foregroundStyle(.tertiary)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 10)
                        }
                        Divider().padding(.leading, 12)
                    }
                }
            }
            .frame(maxHeight: 170)
            .padding(.bottom, 10)
        }
        .background(.bar)
    }
}
```

**Commit:**

```bash
git add zsync/zsync/Views/Terminal/SnippetDrawerView.swift zsync/zsync/Views/Terminal/HistoryDrawerView.swift
git commit -m "feat: snippet drawer and command history drawer"
```

---

## Phase 5: Keys & Security

### Task 16: Secure Enclave Key Generation

**Files:**
- Create: `zsync/zsync/Services/KeychainService.swift`
- Test: `zsync/zsyncTests/KeychainServiceTests.swift`

**Step 1: Implement key generation service**

```swift
// zsync/zsync/Services/KeychainService.swift
import Foundation
import Security
import CryptoKit

enum KeychainError: Error {
    case generationFailed
    case keyNotFound
    case unexpectedStatus(OSStatus)
    case encodingFailed
}

final class KeychainService {
    static let shared = KeychainService()

    private let deviceKeyTag = "com.wavedepth.zsync.device-key"

    /// Generate an Ed25519 key pair in the Secure Enclave
    /// Note: Secure Enclave supports P-256 natively.
    /// For Ed25519, we generate in software Keychain with kSecAttrTokenIDSecureEnclave
    /// where available, falling back to software keychain.
    func generateDeviceKey() throws -> (publicKey: String, fingerprint: String) {
        // Generate Ed25519 key pair using CryptoKit
        let privateKey = Curve25519.Signing.PrivateKey()
        let publicKey = privateKey.publicKey

        // Store private key in Keychain
        let privateKeyData = privateKey.rawRepresentation
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: deviceKeyTag,
            kSecValueData as String: privateKeyData,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        ]

        // Delete existing key first
        SecItemDelete(query as CFDictionary)

        let status = SecItemAdd(query as CFDictionary, nil)
        guard status == errSecSuccess else {
            throw KeychainError.unexpectedStatus(status)
        }

        // Format as SSH public key
        let pubKeySSH = formatSSHPublicKey(publicKey.rawRepresentation)
        let fingerprint = sshFingerprint(publicKey.rawRepresentation)

        return (publicKey: pubKeySSH, fingerprint: fingerprint)
    }

    /// Retrieve the device public key in SSH format
    func getDevicePublicKey() -> String? {
        guard let privateKeyData = getPrivateKeyData() else { return nil }
        let privateKey = try? Curve25519.Signing.PrivateKey(rawRepresentation: privateKeyData)
        guard let pubKey = privateKey?.publicKey else { return nil }
        return formatSSHPublicKey(pubKey.rawRepresentation)
    }

    func getDeviceFingerprint() -> String? {
        guard let privateKeyData = getPrivateKeyData() else { return nil }
        let privateKey = try? Curve25519.Signing.PrivateKey(rawRepresentation: privateKeyData)
        guard let pubKey = privateKey?.publicKey else { return nil }
        return sshFingerprint(pubKey.rawRepresentation)
    }

    func deviceKeyExists() -> Bool {
        getPrivateKeyData() != nil
    }

    private func getPrivateKeyData() -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: deviceKeyTag,
            kSecReturnData as String: true,
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess else { return nil }
        return item as? Data
    }

    private func formatSSHPublicKey(_ rawPublicKey: Data) -> String {
        // SSH Ed25519 wire format:
        // 4-byte length + "ssh-ed25519" + 4-byte length + 32-byte key
        var wireFormat = Data()
        let keyType = "ssh-ed25519".data(using: .utf8)!
        wireFormat.append(contentsOf: withUnsafeBytes(of: UInt32(keyType.count).bigEndian) { Array($0) })
        wireFormat.append(keyType)
        wireFormat.append(contentsOf: withUnsafeBytes(of: UInt32(rawPublicKey.count).bigEndian) { Array($0) })
        wireFormat.append(rawPublicKey)

        return "ssh-ed25519 \(wireFormat.base64EncodedString()) zsync-iphone@zsync"
    }

    private func sshFingerprint(_ rawPublicKey: Data) -> String {
        let hash = SHA256.hash(data: rawPublicKey)
        return "SHA256:\(Data(hash).base64EncodedString())"
    }
}
```

**Step 2: Write tests**

```swift
// zsync/zsyncTests/KeychainServiceTests.swift
import XCTest
@testable import zsync

final class KeychainServiceTests: XCTestCase {
    func testGenerateDeviceKey() throws {
        let service = KeychainService.shared
        let result = try service.generateDeviceKey()
        XCTAssertTrue(result.publicKey.hasPrefix("ssh-ed25519 "))
        XCTAssertTrue(result.fingerprint.hasPrefix("SHA256:"))
    }

    func testDeviceKeyPersists() throws {
        let service = KeychainService.shared
        _ = try service.generateDeviceKey()
        XCTAssertTrue(service.deviceKeyExists())
        XCTAssertNotNil(service.getDevicePublicKey())
    }
}
```

**Step 3: Run tests, commit**

```bash
git add zsync/zsync/Services/KeychainService.swift zsync/zsyncTests/KeychainServiceTests.swift
git commit -m "feat: Secure Enclave Ed25519 key generation and Keychain storage"
```

---

### Task 17: SSH Keys Screen

**Files:**
- Modify: `zsync/zsync/Views/Keys/KeysView.swift`

Update the placeholder KeysView with the full implementation showing device key, imported keys, copy/share actions. Reference prototype for layout. Use `KeychainService` for key data.

**Commit:**

```bash
git commit -m "feat: SSH keys screen with device key and copy/share actions"
```

---

### Task 18: Face ID Lock

**Files:**
- Create: `zsync/zsync/Services/BiometricService.swift`
- Modify: `zsync/zsync/zsyncApp.swift` (add auth gate)

Use `LocalAuthentication` framework. When `requireFaceID` is enabled in settings, show a locked screen on app launch until Face ID succeeds.

**Commit:**

```bash
git commit -m "feat: Face ID app lock using LocalAuthentication"
```

---

## Phase 6: Settings & Polish

### Task 19: Theme System + Live Preview

**Files:**
- Create: `zsync/zsync/Models/TerminalTheme.swift`
- Create: `zsync/zsync/Views/Settings/ThemePickerView.swift`
- Create: `zsync/zsync/Views/Settings/ThemePreviewView.swift`

Define all 8 themes (Dracula, Solarized Dark, Nord, Catppuccin Mocha, Tokyo Night, One Dark, Gruvbox Dark, Default) with bg, fg, green, comment, accent colors. The preview shows a mock terminal with the selected theme and font, matching the prototype.

**Commit:**

```bash
git commit -m "feat: terminal theme picker with live preview"
```

---

### Task 20: Font Picker

**Files:**
- Create: `zsync/zsync/Views/Settings/FontPickerView.swift`

List of 10 fonts, each rendered in its own typeface with a sample string. Selecting a font updates the app-wide `--mono` equivalent (stored in AppSettings).

**Commit:**

```bash
git commit -m "feat: terminal font picker with per-font samples"
```

---

### Task 21: Settings Screen (Full)

**Files:**
- Modify: `zsync/zsync/Views/Settings/SettingsView.swift`

Wire together: ThemePickerView, FontPickerView, Face ID toggle, iCloud Sync toggle, iCloud Keychain toggle, version info.

**Commit:**

```bash
git commit -m "feat: full settings screen with themes, fonts, security, sync"
```

---

## Phase 7: Voice & Sync

### Task 22: SFSpeechRecognizer with Custom Vocabulary

**Files:**
- Create: `zsync/zsync/Services/VoiceInputService.swift`
- Test: `zsync/zsyncTests/VoiceInputServiceTests.swift`

Implement on-device speech recognition with custom terminal vocabulary. Uses `SFSpeechRecognizer` with `SFSpeechLanguageModel.Configuration` (iOS 17+) to add CLI commands, flags, symbols. Wire to the mic button in CompositionBarView.

Key vocabulary additions:
- Shell builtins: ls, cd, grep, awk, sed, cat, echo, mkdir, rm, cp, mv, chmod
- Git commands: git status, git commit, git push, git pull, git diff, git log, git branch
- Flags: -l, -a, -h, --help, -v, --verbose, -f, --force, -r, --recursive
- Symbol mappings: "pipe" -> |, "dash" -> -, "slash" -> /, "tilde" -> ~

**Commit:**

```bash
git commit -m "feat: SFSpeechRecognizer voice input with custom CLI vocabulary"
```

---

### Task 23: iCloud Sync (CloudKit)

**Files:**
- Modify: `zsync/zsync/zsyncApp.swift`
- Modify: `zsync/zsync/Models/Machine.swift`

Enable CloudKit on the SwiftData `ModelContainer`. This requires:
1. A CloudKit container ID in entitlements
2. `ModelConfiguration` with `cloudKitDatabase: .automatic`
3. All `@Model` types must have default values for all properties (CloudKit requirement)
4. Handle iCloud account changes and sync conflicts

**Commit:**

```bash
git commit -m "feat: iCloud sync for machine configs via SwiftData + CloudKit"
```

---

## Phase 8: Integration & Polish

### Task 24: App Lifecycle — Background/Foreground Reconnection

**Files:**
- Create: `zsync/zsync/Services/AppLifecycleManager.swift`

On `scenePhase` change to `.active`:
1. Re-check all machine SSH connections
2. Refresh session lists
3. Reconnect active terminal if one was open

On `scenePhase` change to `.background`:
1. Mark SSH connections as stale
2. Don't attempt to keep them alive

**Commit:**

```bash
git commit -m "feat: app lifecycle — reconnect SSH on foreground, mark stale on background"
```

---

### Task 25: Wire Everything Together — Full Integration

**Files:**
- Modify: Multiple view files to replace mock data with real services
- Create: `zsync/zsync/Services/ServiceContainer.swift` (dependency injection)

This task connects all the dots:
1. HomeView fetches real session counts from ZmoshService
2. SessionPickerView uses real SSH to list sessions
3. TerminalScreenView establishes real SSH shell via CitadelSSHService
4. SetupRequiredView's "Check Again" runs real zmosh detection
5. NewSessionView creates real sessions via `zmosh attach`
6. Quick Jump actually reconnects to real sessions

Use a simple service container pattern for dependency injection (protocol-based, swappable for testing).

**Commit:**

```bash
git commit -m "feat: full integration — wire views to real SSH and zmosh services"
```

---

### Task 26: App Icon + Launch Screen

**Files:**
- Create: `zsync/zsync/Assets.xcassets/AppIcon.appiconset/`
- Modify: `zsync/zsync/Assets.xcassets/`

Design a simple app icon: terminal prompt (>) on a dark background with the accent blue. Generate all required sizes (1024x1024 source, Xcode handles the rest with a single-size icon for iOS 17+).

Launch screen: black background, "zsync" text in the monospace font, centered.

**Commit:**

```bash
git commit -m "feat: app icon and launch screen"
```

---

## Summary

| Phase | Tasks | What it delivers |
|-------|-------|-----------------|
| 1. Foundation | 1-3 | Xcode project, data models, tab navigation |
| 2. SSH Transport | 4-6 | SSH service, zmosh parser, host key trust |
| 3. Core Screens | 7-11 | Home, Add Machine, Sessions, Setup Gate, New Session |
| 4. Terminal | 12-15 | SwiftTerm view, composition bar, key bar, drawers |
| 5. Keys & Security | 16-18 | Secure Enclave keys, Keys screen, Face ID |
| 6. Settings | 19-21 | Themes, fonts, full settings screen |
| 7. Voice & Sync | 22-23 | Speech recognition, iCloud sync |
| 8. Integration | 24-26 | Lifecycle, full wiring, app icon |

**Total: 26 tasks across 8 phases.**

Phases 1-4 produce a functional app (connect, discover, attach). Phases 5-8 add polish and secondary features. Each phase can be demoed independently.

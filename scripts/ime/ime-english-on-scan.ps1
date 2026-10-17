#Requires -Version 5.1
<#
  扫描页输入法助手
  ================

  作用：当浏览器停在「出入库」页、且光标在扫码框里时，把 Windows 输入法切到英文；
  光标离开扫码框（例如去填备注、或者切到别的程序）就停止干预。

  为什么需要它：网页**无法**直接切换操作系统的输入法（浏览器没有这个 API）。
  所以扫码页把自己"焦点在扫码框"这件事通过窗口标题里的一个零宽字符发布出来，
  这个脚本轮询前台窗口标题，看到标记就调用 ImmSetConversionStatus 切到英文。

  用法：
      start-ime-helper.cmd
  或：
      powershell -ExecutionPolicy Bypass -File .\ime-english-on-scan.ps1
  Ctrl+C 停止。

  注意：
  - 不需要的时候不要开；它只影响"当前前台窗口 + 扫码框聚焦"这一种情况。
  - 如果你用的不是 Chrome/Edge/Firefox 等常见浏览器，或者输入法自己不接受这个调用，
    它可能不生效——此时页面本身仍会检测到中文输入法并丢弃那次输入，不会写错数据。
#>
param(
    [int]$PollMilliseconds = 200
)

$ErrorActionPreference = 'Stop'
$marker = [string][char]0x200B

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Text;

public static class ScanIme
{
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int Left, Top, Right, Bottom; }

    [StructLayout(LayoutKind.Sequential)]
    public struct GUITHREADINFO
    {
        public int cbSize;
        public int flags;
        public IntPtr hwndActive;
        public IntPtr hwndFocus;
        public IntPtr hwndCapture;
        public IntPtr hwndMenuOwner;
        public IntPtr hwndMoveSize;
        public IntPtr hwndCaret;
        public RECT rcCaret;
    }

    [DllImport("user32.dll")]
    public static extern IntPtr GetForegroundWindow();

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int maxCount);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);

    [DllImport("user32.dll")]
    public static extern bool GetGUIThreadInfo(uint threadId, ref GUITHREADINFO info);

    [DllImport("imm32.dll")]
    public static extern IntPtr ImmGetContext(IntPtr hWnd);

    [DllImport("imm32.dll")]
    public static extern bool ImmSetConversionStatus(IntPtr himc, int conversion, int sentence);

    [DllImport("imm32.dll")]
    public static extern bool ImmReleaseContext(IntPtr hWnd, IntPtr himc);

    public static string Title(IntPtr hWnd)
    {
        var sb = new StringBuilder(512);
        GetWindowText(hWnd, sb, sb.Capacity);
        return sb.ToString();
    }

    // Switches the focused control of the foreground window to alphanumeric
    // (English) mode. Returns true when the call succeeded.
    public static bool ForceEnglish(IntPtr hWnd)
    {
        uint pid;
        uint threadId = GetWindowThreadProcessId(hWnd, out pid);

        IntPtr target = hWnd;
        var info = new GUITHREADINFO();
        info.cbSize = Marshal.SizeOf(typeof(GUITHREADINFO));
        if (GetGUIThreadInfo(threadId, ref info) && info.hwndFocus != IntPtr.Zero)
            target = info.hwndFocus;

        IntPtr himc = ImmGetContext(target);
        if (himc == IntPtr.Zero) return false;

        // IME_CMODE_ALPHANUMERIC = 0x0000
        bool ok = ImmSetConversionStatus(himc, 0, 0);
        ImmReleaseContext(target, himc);
        return ok;
    }
}
'@

Write-Host "扫描页输入法助手已启动（Ctrl+C 停止）" -ForegroundColor Green
Write-Host "只在扫码框获得焦点时把输入法切到英文；点其它输入框或其它窗口不受影响。" -ForegroundColor Gray

$lastActive = $false
while ($true) {
    $active = $false
    try {
        $hwnd = [ScanIme]::GetForegroundWindow()
        if ($hwnd -ne [IntPtr]::Zero) {
            $title = [ScanIme]::Title($hwnd)
            if ($title.StartsWith($marker)) {
                $active = $true
                [void][ScanIme]::ForceEnglish($hwnd)
            }
        }
    } catch {
        # A window can disappear between the two calls above; just retry.
    }

    if ($active -ne $lastActive) {
        $lastActive = $active
        if ($active) {
            Write-Host "扫码框已聚焦 -> 输入法切到英文" -ForegroundColor Cyan
        } else {
            Write-Host "扫码框已离开 -> 不再干预输入法" -ForegroundColor DarkGray
        }
    }

    Start-Sleep -Milliseconds $PollMilliseconds
}

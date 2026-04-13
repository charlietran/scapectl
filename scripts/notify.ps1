# notify.ps1 — Show a Windows toast notification (Windows 10+).
# Shipped with scapectl for use in trigger scripts.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File notify.ps1 -Message "Headset connected"
#   powershell -ExecutionPolicy Bypass -File notify.ps1 -Title "Scape" -Message "Battery low"

param(
    [string]$Title   = "Scape",
    [string]$Message = ""
)

[void][Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]

$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent(
    [Windows.UI.Notifications.ToastTemplateType]::ToastText02
)
$nodes = $template.GetElementsByTagName("text")
$nodes.Item(0).AppendChild($template.CreateTextNode($Title))   | Out-Null
$nodes.Item(1).AppendChild($template.CreateTextNode($Message)) | Out-Null

[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("ScapeCtl").Show(
    [Windows.UI.Notifications.ToastNotification]::new($template)
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()
$form = New-Object System.Windows.Forms.Form
$form.Text = 'Pulse Server — создание администратора'
$form.ClientSize = New-Object System.Drawing.Size(520, 335)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false
$form.MinimizeBox = $false
$form.AutoScaleMode = 'Dpi'

$info = New-Object System.Windows.Forms.Label
$info.Text = "Первый запуск сервера. Задайте пароль администратора.`nЛогин — ADMIN_LOGIN, если задан; иначе admin."
$info.SetBounds(20, 15, 480, 45)
$form.Controls.Add($info)

$label = New-Object System.Windows.Forms.Label
$label.Text = 'Пароль (12–72 байта UTF-8):'
$label.SetBounds(20, 65, 470, 22)
$form.Controls.Add($label)
$first = New-Object System.Windows.Forms.TextBox
$first.Name = 'Password'
$first.UseSystemPasswordChar = $true
$first.ShortcutsEnabled = $true
$first.SetBounds(20, 90, 480, 28)
$form.Controls.Add($first)

$label2 = New-Object System.Windows.Forms.Label
$label2.Text = 'Повторите пароль:'
$label2.SetBounds(20, 130, 470, 22)
$form.Controls.Add($label2)
$second = New-Object System.Windows.Forms.TextBox
$second.Name = 'Confirmation'
$second.UseSystemPasswordChar = $true
$second.ShortcutsEnabled = $true
$second.SetBounds(20, 155, 480, 28)
$form.Controls.Add($second)

$show = New-Object System.Windows.Forms.CheckBox
$show.Text = 'Показать пароль'
$show.SetBounds(20, 195, 260, 25)
$show.Add_CheckedChanged({
    $first.UseSystemPasswordChar = -not $show.Checked
    $second.UseSystemPasswordChar = -not $show.Checked
})
$form.Controls.Add($show)
$errorLabel = New-Object System.Windows.Forms.Label
$errorLabel.ForeColor = [System.Drawing.Color]::Firebrick
$errorLabel.SetBounds(20, 225, 480, 40)
$form.Controls.Add($errorLabel)

$save = New-Object System.Windows.Forms.Button
$save.Text = 'Создать и запустить'
$save.SetBounds(20, 280, 245, 32)
$save.Add_Click({
    $length = [System.Text.Encoding]::UTF8.GetByteCount($first.Text)
    if ($length -lt 12 -or $length -gt 72) {
        $errorLabel.Text = 'Нужно от 12 до 72 байт UTF-8 (для кириллицы — 2 байта на букву).'
        $first.Focus()
        return
    }
    if ($first.Text -cne $second.Text) {
        $errorLabel.Text = 'Пароли не совпадают. Повторите ввод.'
        $second.Focus()
        return
    }
    $form.DialogResult = [System.Windows.Forms.DialogResult]::OK
    $form.Close()
})
$form.Controls.Add($save)
$cancel = New-Object System.Windows.Forms.Button
$cancel.Text = 'Отмена'
$cancel.SetBounds(355, 280, 145, 32)
$cancel.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
$form.Controls.Add($cancel)
$form.AcceptButton = $save
$form.CancelButton = $cancel
$form.Add_Shown({ $form.Activate(); $first.Select() })
try {
    if ($form.ShowDialog() -ne [System.Windows.Forms.DialogResult]::OK) { exit 2 }
    # Base64 is a transport encoding only, not encryption. No password files/logs/argv.
    [Console]::Out.Write([Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($first.Text)))
} finally {
    $first.Clear()
    $second.Clear()
    $form.Dispose()
}

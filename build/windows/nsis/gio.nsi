Unicode true

####
## Installer of LeanTerm (cmd/leanterm) for Windows, amd64.
## build/package-windows.sh calls it with:
##   makensis -DINFO_PRODUCTVERSION=2.1.3 -DARG_BINARY=leanterm.exe -DARG_LEANEDIT=leanedit.exe -DOUTFILE=setup.exe gio.nsi
## The installer is per machine (Program Files, administrator rights). The
## uninstall key keeps its name, so an update replaces an installed version.
####

!ifndef INFO_PRODUCTVERSION
    !define INFO_PRODUCTVERSION "2.1.3"
!endif
!ifndef ARG_BINARY
    !error "ARG_BINARY must name the leanterm.exe to install"
!endif
!ifndef ARG_LEANEDIT
    !error "ARG_LEANEDIT must name the leanedit.exe to install"
!endif
!ifndef OUTFILE
    !define OUTFILE "..\..\..\bin\leanterm_setup.exe"
!endif

!define INFO_PRODUCTNAME "LeanTerm"
!define INFO_COMPANYNAME "LeanTerm Project"
!define INFO_COPYRIGHT   "© 2026 LeanTerm Project, GPL-3.0"
!define PRODUCT_EXECUTABLE "leanterm.exe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"

# LeanEdit: LeanTerm's text editor on its own, with an executable of its own
# (icon, name in "Open with" and the taskbar).
!define LEANEDIT_EXECUTABLE "leanedit.exe"
!define LEANEDIT_PROGID "LeanTerm.LeanEdit"
!define LEANEDIT_CMD '"$INSTDIR\${LEANEDIT_EXECUTABLE}"'
# The AppUserModelIDs of the windows (see internal/gui/winsys): shortcuts
# carry them too, so a pinned shortcut and its windows share a button.
!define LEANTERM_APPID "org.leanterm.LeanTerm"
!define LEANEDIT_APPID "org.leanterm.LeanEdit"

RequestExecutionLevel admin

# The version information must consist of 4 parts.
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
!include "x64.nsh"
!include "WinVer.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can read the details.
!define MUI_ABORTWARNING # Warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME
!define MUI_COMPONENTSPAGE_SMALLDESC
!insertmacro MUI_PAGE_COMPONENTS # LeanEdit is optional.
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "${OUTFILE}"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
    ${IfNot} ${AtLeastWin10}
        MessageBox MB_OK "LeanTerm needs Windows 10 or later."
        Abort
    ${EndIf}
    ${IfNot} ${RunningX64}
        MessageBox MB_OK "LeanTerm is built for 64-bit Windows only."
        Abort
    ${EndIf}
FunctionEnd

Section "LeanTerm" SecLeanTerm
    SectionIn RO # always installed
    SetShellVarContext all

    SetOutPath $INSTDIR
    File "/oname=${PRODUCT_EXECUTABLE}" "${ARG_BINARY}"
    File "..\leanedit.ico"

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortcut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" -set-shortcut-appid ${LEANTERM_APPID} "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$DESKTOP\${INFO_PRODUCTNAME}.lnk"'

    WriteUninstaller "$INSTDIR\uninstall.exe"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
    WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
SectionEnd

####
## LeanEdit: its own shortcuts and "Open with" for text files.
####
!macro leanedit.openWith EXT
    WriteRegStr HKLM "Software\Classes\${EXT}\OpenWithProgids" "${LEANEDIT_PROGID}" ""
!macroend
!macro leanedit.removeOpenWith EXT
    DeleteRegValue HKLM "Software\Classes\${EXT}\OpenWithProgids" "${LEANEDIT_PROGID}"
!macroend
# Text file types offered "Open with LeanEdit" (used for install and uninstall).
!macro leanedit.forEachExtension MACRO
    !insertmacro ${MACRO} ".txt"
    !insertmacro ${MACRO} ".log"
    !insertmacro ${MACRO} ".md"
    !insertmacro ${MACRO} ".ini"
    !insertmacro ${MACRO} ".cfg"
    !insertmacro ${MACRO} ".conf"
    !insertmacro ${MACRO} ".config"
    !insertmacro ${MACRO} ".json"
    !insertmacro ${MACRO} ".xml"
    !insertmacro ${MACRO} ".yaml"
    !insertmacro ${MACRO} ".yml"
    !insertmacro ${MACRO} ".toml"
    !insertmacro ${MACRO} ".csv"
    !insertmacro ${MACRO} ".properties"
    !insertmacro ${MACRO} ".env"
    !insertmacro ${MACRO} ".sql"
    !insertmacro ${MACRO} ".sh"
    !insertmacro ${MACRO} ".ps1"
    !insertmacro ${MACRO} ".bat"
    !insertmacro ${MACRO} ".cmd"
    !insertmacro ${MACRO} ".py"
    !insertmacro ${MACRO} ".js"
    !insertmacro ${MACRO} ".ts"
    !insertmacro ${MACRO} ".go"
    !insertmacro ${MACRO} ".c"
    !insertmacro ${MACRO} ".h"
    !insertmacro ${MACRO} ".cpp"
    !insertmacro ${MACRO} ".cs"
    !insertmacro ${MACRO} ".java"
    !insertmacro ${MACRO} ".css"
    !insertmacro ${MACRO} ".html"
    !insertmacro ${MACRO} ".htm"
!macroend

Section "LeanEdit (text editor)" SecLeanEdit
    SetShellVarContext all

    SetOutPath $INSTDIR
    File "/oname=${LEANEDIT_EXECUTABLE}" "${ARG_LEANEDIT}"

    CreateShortcut "$SMPROGRAMS\LeanEdit.lnk" "$INSTDIR\${LEANEDIT_EXECUTABLE}" "" "$INSTDIR\${LEANEDIT_EXECUTABLE}" 0 "" "" "LeanEdit text editor"
    CreateShortcut "$DESKTOP\LeanEdit.lnk" "$INSTDIR\${LEANEDIT_EXECUTABLE}" "" "$INSTDIR\${LEANEDIT_EXECUTABLE}" 0 "" "" "LeanEdit text editor"
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" -set-shortcut-appid ${LEANEDIT_APPID} "$SMPROGRAMS\LeanEdit.lnk" "$DESKTOP\LeanEdit.lnk"'

    # ProgID used by "Open with": name and icon of LeanEdit, opens the file in it.
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}" "" "Text Document (LeanEdit)"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}" "FriendlyTypeName" "Text Document (LeanEdit)"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}" "AppUserModelID" "${LEANEDIT_APPID}"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\DefaultIcon" "" "$INSTDIR\${LEANEDIT_EXECUTABLE},0"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\Application" "ApplicationName" "LeanEdit"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\Application" "ApplicationDescription" "LeanEdit text editor (LeanTerm)"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\Application" "ApplicationIcon" "$INSTDIR\${LEANEDIT_EXECUTABLE},0"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\Application" "AppUserModelID" "${LEANEDIT_APPID}"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\shell\open" "FriendlyAppName" "LeanEdit"
    WriteRegStr HKLM "Software\Classes\${LEANEDIT_PROGID}\shell\open\command" "" '${LEANEDIT_CMD} "%1"'
    !insertmacro leanedit.forEachExtension leanedit.openWith
    # The application itself, for "Open with" → "Choose another app".
    WriteRegStr HKLM "Software\Classes\Applications\${LEANEDIT_EXECUTABLE}" "FriendlyAppName" "LeanEdit"
    WriteRegStr HKLM "Software\Classes\Applications\${LEANEDIT_EXECUTABLE}\DefaultIcon" "" "$INSTDIR\${LEANEDIT_EXECUTABLE},0"
    WriteRegStr HKLM "Software\Classes\Applications\${LEANEDIT_EXECUTABLE}\shell\open\command" "" '${LEANEDIT_CMD} "%1"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\${LEANEDIT_EXECUTABLE}" "" "$INSTDIR\${LEANEDIT_EXECUTABLE}"
    # Any other file Windows considers text (perceived type "text").
    WriteRegStr HKLM "Software\Classes\SystemFileAssociations\text\OpenWithProgids" "${LEANEDIT_PROGID}" ""

    # Tell Explorer about the new "Open with" entries.
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecLeanTerm} "LeanTerm connection manager (required)."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecLeanEdit} "LeanEdit, LeanTerm's text editor on its own: Start menu and desktop shortcut and $\"Open with$\" for text files."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "un.Uninstall"
    SetShellVarContext all

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    # LeanEdit (nothing to do when it was not selected)
    Delete "$SMPROGRAMS\LeanEdit.lnk"
    Delete "$DESKTOP\LeanEdit.lnk"
    !insertmacro leanedit.forEachExtension leanedit.removeOpenWith
    DeleteRegValue HKLM "Software\Classes\SystemFileAssociations\text\OpenWithProgids" "${LEANEDIT_PROGID}"
    DeleteRegKey HKLM "Software\Classes\${LEANEDIT_PROGID}"
    DeleteRegKey HKLM "Software\Classes\Applications\${LEANEDIT_EXECUTABLE}"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\${LEANEDIT_EXECUTABLE}"
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'

    RMDir /r $INSTDIR

    DeleteRegKey HKLM "${UNINST_KEY}"
SectionEnd

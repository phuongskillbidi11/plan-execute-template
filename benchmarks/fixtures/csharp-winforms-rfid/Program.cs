using System;
using System.Windows.Forms;

namespace RfidConfigTool
{
    // Minimal fixture stub representing a C# WinForms UHF RFID configuration
    // tool that talks to hardware over RS485/USB-HID — see Phase 9
    // benchmarks/scenarios/csharp-winforms-rs485.yaml.
    internal static class Program
    {
        [STAThread]
        static void Main()
        {
            Application.Run(new MainForm());
        }
    }

    internal class MainForm : Form
    {
    }
}

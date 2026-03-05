//------------------------------------------------------------------------------
// TribalOutpostAutoDL.cs
// TribalOutpost AutoDownload - File-based protocol
//
// Based on the original BytAutoDownload.cs by Ross A. Carlson (Bytor), 2001
// Updated for TribalOutpost.com with file-based IPC protocol.
//
// Instead of downloading via HTTP+UUencode, this script writes a request file
// that the companion app (to-autodownload) watches for. The companion app
// downloads the VL2 over HTTPS and writes a response file when complete.
//------------------------------------------------------------------------------

// Configuration variables.
$TOADDebugLevel = 0;
$TOADGuiScript = "scripts/autoexec/tribaloutpost/gui/TribalOutpostAutoDL.gui";
$TOADWorkingDir = "TribalOutpostAutoDL";
$TOADRequestFile = $TOADWorkingDir @ "/request.adl";
$TOADResponseFile = $TOADWorkingDir @ "/response.adl";
$TOADPollTimeout = 120;  // seconds to wait for companion app

activatePackage (TribalOutpostADL);

//------------------------------------------------------------------------------

package TribalOutpostADL {

	function MessageBoxOKDlg::onWake (%this) {
		parent::onWake( %this );

		%message = MBOKText.getText();

		// Strip off the center justification tag.
		%message = getSubStr (%message, 13, 1000);

		// Detect missing map file errors.
		if (getSubStr (%message, 0, 24) $= "Unable to load interior:") {
			%missingFile = getSubStr (%message, 25, 1000);
			TOADShowInitialPrompt (%missingFile);
		} else if (getSubStr (%message, 0, 51) $= "You are missing a file needed to play this mission:") {
			%missingFile = getSubStr (%message, 52, 1000);
			TOADShowInitialPrompt (%missingFile);
		}
	}

	//--------------------------------------------------------------------------

	function JoinGame (%address) {
		$TOADLastServerAddress = %address;
		$TOADMissionDisplayName = "";
		parent::JoinGame (%address);
	}

	//--------------------------------------------------------------------------

	function handleLoadInfoMessage (%msgType, %msgString, %bitmapName, %mapName, %missionType) {
		// Only capture the map name when a mission type is present (e.g. "CTF",
		// "Siege"). Mod/game-type info rotates through the same callback but
		// without a mission type, so we skip those to avoid overwriting the
		// real map name with something like "Classic 1.1".
		if (%missionType !$= "") {
			$TOADMissionDisplayName = %mapName;
		}
		parent::handleLoadInfoMessage (%msgType, %msgString, %bitmapName, %mapName, %missionType);
	}
}; // End package TribalOutpostADL.

//------------------------------------------------------------------------------

function TOADShowInitialPrompt (%missingFile) {
	TOADEcho ("Missing file detected: " @ %missingFile, 1);

	// Close the generic error dialog.
	Canvas.popDialog (MessageBoxOKDlg);

	// Play a sound.
	%handle = alxCreateSource (AudioGui, "fx/misc/diagnostic_beep.wav");
	alxPlay (%handle);

	// Load the GUI.
	if (! isObject (TOADInitialPromptDlg)) {
		exec ($TOADGuiScript);
	}

	// Show the prompt dialog.
	%message = "<just:center><spush><font:copperplate gothic bold:18><color:FFFFFF>Missing Map: " @ $TOADMissionDisplayName @ "<spop>\n\nThe server you are trying to join is running a map you don't have. Would you like to download it from TribalOutpost.com?\n\n<font:arial:12><color:AAAAAA>Make sure the AutoDownload companion app is running.";
	TOADInitialPromptText.setText (%message);
	TOADInitialPromptFileText.setText ("<color:555555><just:center>Missing file: " @ %missingFile);
	$TOADMissingFile = %missingFile;
	Canvas.pushDialog (TOADInitialPromptDlg);
}

//------------------------------------------------------------------------------

function TOADInitiateDownload () {
	TOADEcho ("Initiating download request", 1);

	// Remove the prompt dialog.
	Canvas.popDialog (TOADInitialPromptDlg);

	// Initialize state.
	$TOADPollCount = 0;
	$TOADDownloadCancelled = 0;
	$TOADTransferError = 0;

	// Ensure the working directory exists.
	// FileObject will create parent dirs on write.

	// Note: stale response files are cleaned up by the companion app when it
	// detects a new request. We do NOT create an empty file here because the
	// poll loop would immediately detect it as a (empty/invalid) response.

	// Write the request file for the companion app.
	%reqFile = new FileObject();
	%reqFile.openForWrite ($TOADRequestFile);
	%reqFile.writeLine ("display_name=" @ $TOADMissionDisplayName);
	%reqFile.writeLine ("filename=" @ $TOADMissingFile);
	%reqFile.close();
	%reqFile.delete();

	TOADEcho ("Request file written: " @ $TOADRequestFile, 1);

	// Show the progress dialog.
	TOADProgressFrame.setText ("AutoDownload");
	TOADProgressStatusText.setText ("Waiting for companion app to download " @ $TOADMissionDisplayName @ " ...");
	TOADProgressText.setText ("Downloading " @ $TOADMissionDisplayName);
	TOADSetProgressValue (0);
	TOADProgressBytesText.setText ("");
	TOADProgressTimeRemText.setText ("");
	Canvas.pushDialog (TOADProgressDlg);

	// Start polling for the response.
	schedule (1000, 0, TOADPollResponse);
}

//------------------------------------------------------------------------------

function TOADSetProgressValue (%val) {
	TOADProgress.setValue (%val);
}

//------------------------------------------------------------------------------

function TOADNoDownload () {
	TOADEcho ("User declined download", 1);
	Canvas.popDialog (TOADInitialPromptDlg);
}

//------------------------------------------------------------------------------

function TOADCancelDownload () {
	TOADEcho ("Download cancelled", 1);
	if ($TOADTransferError) {
		TOADCloseProgressDialog ();
	} else if (! $TOADDownloadCancelled) {
		TOADProgressStatusText.setText ("Download cancelled!");
		$TOADDownloadCancelled = 1;
		schedule (1000, 0, TOADCloseProgressDialog);
	}
}

//------------------------------------------------------------------------------

function TOADCloseProgressDialog () {
	Canvas.popDialog (TOADProgressDlg);
}

//------------------------------------------------------------------------------

function TOADPollResponse () {
	if ($TOADDownloadCancelled) {
		return;
	}

	$TOADPollCount ++;
	TOADEcho ("Polling for response ... " @ $TOADPollCount, 2);

	// Animate the progress bar to show we're alive
	%pulse = ($TOADPollCount % 20) / 20.0;
	TOADSetProgressValue (%pulse);
	TOADProgressTimeRemText.setText ("Waiting: " @ $TOADPollCount @ "s / " @ $TOADPollTimeout @ "s");

	// Check if we've been waiting too long.
	if ($TOADPollCount > $TOADPollTimeout) {
		error ("AutoDownload: response timed out!");
		%handle = alxCreateSource (AudioGui, "fx/misc/warning_beep.wav");
		alxPlay (%handle);
		$TOADTransferError = 1;
		TOADProgressFrame.setText ("ERROR!");
		TOADCancelButton.setValue ("OK");
		TOADProgressStatusText.setText ("<color:FF2011>Timed out waiting for companion app. Make sure to-autodownload is running.");
		return;
	}

	// Check for response file using FileObject (isFile uses cached VFS and won't
	// detect files created externally by the companion app).
	// We also verify the file has a "status=" line to avoid acting on empty or
	// partially-written files.
	%checkFile = new FileObject();
	%opened = %checkFile.openForRead ($TOADResponseFile);
	if (%opened) {
		%hasStatus = 0;
		while (! %checkFile.isEOF()) {
			%checkLine = %checkFile.readLine();
			if (getSubStr (%checkLine, 0, 7) $= "status=") {
				%hasStatus = 1;
			}
		}
		%checkFile.close();
		%checkFile.delete();
		if (%hasStatus) {
			TOADEcho ("Response file found!", 1);
			TOADHandleResponse ();
			return;
		} else {
			TOADEcho ("Response file exists but has no status, ignoring", 2);
		}
	} else {
		%checkFile.close();
		%checkFile.delete();
	}

	// Keep polling.
	schedule (1000, 0, TOADPollResponse);
}

//------------------------------------------------------------------------------

function TOADHandleResponse () {
	// Read the response file.
	%respFile = new FileObject();
	%respFile.openForRead ($TOADResponseFile);

	%status = "";
	%vl2 = "";
	%message = "";

	while (! %respFile.isEOF()) {
		%line = %respFile.readLine();
		if (getSubStr (%line, 0, 7) $= "status=") {
			%status = getSubStr (%line, 7, 1000);
		} else if (getSubStr (%line, 0, 4) $= "vl2=") {
			%vl2 = getSubStr (%line, 4, 1000);
		} else if (getSubStr (%line, 0, 8) $= "message=") {
			%message = getSubStr (%line, 8, 1000);
		}
	}

	%respFile.close();
	%respFile.delete();

	TOADEcho ("Response status: " @ %status @ ", vl2: " @ %vl2 @ ", message: " @ %message, 1);

	if (%status $= "ok") {
		TOADShowRejoinPrompt (%vl2);
	} else {
		// Close the progress dialog and show an error dialog.
		Canvas.popDialog (TOADProgressDlg);

		%handle = alxCreateSource (AudioGui, "fx/misc/warning_beep.wav");
		alxPlay (%handle);

		if (%message $= "") {
			%message = "Download failed. Unknown error.";
		}

		error ("AutoDownload: " @ %message);
		MessageBoxOK ("AutoDownload Error", %message);
	}
}

//------------------------------------------------------------------------------

function TOADShowRejoinPrompt (%vl2) {
	TOADEcho ("Download complete: " @ %vl2, 1);
	Canvas.popDialog (TOADProgressDlg);

	// Rebuild mod paths so T2 sees the new VL2.
	rebuildModPaths ();
	buildMissionList ();

	%yesCallback = "TOADRejoinServer ();";
	%noCallback = "TOADNoRejoinServer ();";
	MessageBoxYesNo ("COMPLETE", "The map has been downloaded (" @ %vl2 @ "). Would you like to rejoin the game server?", %yesCallback, %noCallback);
	%handle = alxCreateSource (AudioGui, "fx/misc/hunters_horde.wav");
	alxPlay (%handle);
}

//------------------------------------------------------------------------------

function TOADRejoinServer () {
	TOADEcho ("Rejoining server: " @ $TOADLastServerAddress, 1);
	rebuildModPaths ();
	buildMissionList ();
	Canvas.popDialog (MessageBoxYesNoDlg);
	disconnect ();
	schedule (2000, 0, "JoinGame", $TOADLastServerAddress);
}

//------------------------------------------------------------------------------

function TOADNoRejoinServer () {
	TOADEcho ("User declined rejoin", 1);
	rebuildModPaths ();
	buildMissionList ();
	Canvas.popDialog (MessageBoxYesNoDlg);
}

//------------------------------------------------------------------------------

function TOADEcho (%message, %debugLevel) {
	if ($TOADDebugLevel >= %debugLevel) {
		echo ("TOAutoDownload: " @ %message);
	}
}

//------------------------------------------------------------------------------

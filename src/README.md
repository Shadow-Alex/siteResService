web driver startup sequence:
1. start "startx" first: nohup startx &
2. export DISPLAY: export DISPLAY=:1  # note that this value should be check first, not always 1
3. start selenium-server-standalone at vender dir: nohup java -Dwebdriver.gecko.driver=geckodriver -cp selenium-server-standalone-3.141.59.jar org.openqa.grid.selenium.GridLauncherV3 -port 8083 &

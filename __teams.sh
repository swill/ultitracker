source read_ini.sh

read_ini config.toml --prefix toml

for team_str in ${toml__ALL_SECTIONS}
do
	team=${team_str#team_}
	color="toml__${team_str}__color"
	# rebuild the theme with the appropriate colors
	echo "building ${team} theme..."
	lessc --modify-var="light-blue=${!color};" --clean-css build/skin-tracker.less static/team/${team}/css/skin-tracker.min.css
	lessc --modify-var="light-blue=${!color};" --clean-css build/AdminLTE/build/less/AdminLTE.less static/team/${team}/css/AdminLTE.min.css
done


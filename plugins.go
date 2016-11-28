package main

// mindl - A downloader for various sites and services.
// Copyright (C) 2016  Mino <mino@minomino.org>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import (
	"github.com/MinoMino/mindl/plugins"
	"github.com/MinoMino/mindl/plugins/booklive"
	"github.com/MinoMino/mindl/plugins/dummy"
	ebj "github.com/MinoMino/mindl/plugins/ebookjapan"
)

// Global slice of Plugin objects. As much as I'd love
// to be able to omit this, Go just doesn't let me for now.
// Could explore using the -X linker flag to set this.
var Plugins = [...]plugins.Plugin{
	&dummy.Plugin,
	&booklive.Plugin,
	&ebj.Plugin,
}

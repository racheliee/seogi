//
//  ContentView.swift
//  mac-app
//
//  Created by rachelie on 2/25/25.
//

import SwiftUI

struct ContentView: View {
    @State private var password: String = ""
    @State private var repeatPassword: String = ""
    @State private var deleteOriginalFile: Bool = false
    @State private var selectedTemplate = "None"
    
    let templateOptions = ["CV", "Report", "Assignment", "Custom"]
    
    var body: some View {
        VStack (spacing: 15){
            Text("Conversion Settings").font(.title2).bold()
            HStack {
                Text("Password:")
                SecureField("", text: $password)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .frame(width: 150)
                Button(action: { password = "" }) {
                    Image(systemName: "eye.slash")
                }
            }
            
            HStack {
                Text("Repeat:")
                SecureField("", text: $repeatPassword)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .frame(width: 150)
            }
            
            HStack {
                Text("Template:")
                Picker("Select template", selection: $selectedTemplate) {
                    ForEach(templateOptions, id: \.self) { size in
                        Text(size)
                    }
                }
                .pickerStyle(MenuPickerStyle())
                .frame(width: 120)
            }
            
            Text("Selected Template: \(selectedTemplate)")
                .font(.caption)
                .foregroundColor(.gray)
        }
        .padding()
    }
}

#Preview {
    ContentView()
}
